package datacenter

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"
)

func (c *Client) Capabilities() domain.Capabilities {
	c.capabilitiesMu.RLock()
	if c.probedCapabilities != nil {
		capabilities := cloneCapabilities(c.probedCapabilities)
		c.capabilitiesMu.RUnlock()
		return capabilities
	}
	c.capabilitiesMu.RUnlock()
	return c.capabilitiesForVersion("")
}

func (c *Client) GetPullRequest(ctx context.Context, repository domain.Repository, number int) (domain.PullRequest, error) {
	if !c.Capabilities().Supports(domain.CapabilityPullRequests) {
		return domain.PullRequest{}, fmt.Errorf("Data Center does not support %s for this authentication mode", domain.CapabilityPullRequests)
	}
	if err := validateRepository(repository); err != nil || number <= 0 {
		return domain.PullRequest{}, fmt.Errorf("invalid Data Center pull request")
	}
	endpoint := c.repositoryEndpoint(repository, "pull-requests", fmt.Sprint(number))
	var payload pullRequestPayload
	if err := c.getJSON(ctx, &endpoint, &payload); err != nil {
		return domain.PullRequest{}, err
	}
	pullRequest, err := c.mapPullRequest(repository, payload)
	pullRequest.Capabilities = c.Capabilities()
	return pullRequest, err
}

func (c *Client) CreatePullRequest(ctx context.Context, input domain.CreatePullRequestInput) (domain.PullRequest, error) {
	if !c.Capabilities().Supports(domain.CapabilityPullRequests) {
		return domain.PullRequest{}, fmt.Errorf("Data Center does not support %s for this authentication mode", domain.CapabilityPullRequests)
	}
	if err := validateRepository(input.Repository); err != nil || input.Title == "" || input.Source == "" || input.Destination == "" {
		return domain.PullRequest{}, fmt.Errorf("invalid Data Center pull request input")
	}
	endpoint := c.repositoryEndpoint(input.Repository, "pull-requests")
	body := map[string]any{"title": input.Title, "description": input.Description, "fromRef": map[string]string{"id": "refs/heads/" + input.Source}, "toRef": map[string]string{"id": "refs/heads/" + input.Destination}}
	var payload pullRequestPayload
	if err := c.json(ctx, http.MethodPost, &endpoint, body, &payload); err != nil {
		return domain.PullRequest{}, err
	}
	pullRequest, err := c.mapPullRequest(input.Repository, payload)
	pullRequest.Capabilities = c.Capabilities()
	return pullRequest, err
}

func (c *Client) GetReview(ctx context.Context, repository domain.Repository, number int) (domain.Review, error) {
	return c.GetReviewProjected(ctx, repository, number, domain.FullReviewProjection())
}

func (c *Client) GetReviewProjected(ctx context.Context, repository domain.Repository, number int, projection domain.ReviewProjection) (domain.Review, error) {
	if !c.Capabilities().Supports(domain.CapabilityReview) {
		return domain.Review{}, fmt.Errorf("Data Center does not support %s for this authentication mode", domain.CapabilityReview)
	}
	if err := validateRepository(repository); err != nil || number <= 0 {
		return domain.Review{}, fmt.Errorf("invalid Data Center pull request")
	}
	pullRequestEndpoint := c.repositoryEndpoint(repository, "pull-requests", fmt.Sprint(number))
	var payload pullRequestPayload
	if err := c.getJSON(ctx, &pullRequestEndpoint, &payload); err != nil {
		return domain.Review{}, err
	}
	pr, err := c.mapPullRequest(repository, payload)
	if err != nil {
		return domain.Review{}, err
	}
	pr.Capabilities = c.Capabilities()
	diff := ""
	if projection.Diff {
		diffEndpoint := c.repositoryEndpoint(repository, "pull-requests", fmt.Sprint(number)+".diff")
		diff, err = c.getText(ctx, &diffEndpoint)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var files []domain.ReviewFile
	if projection.Files {
		files, err = c.reviewFiles(ctx, repository, number)
		if err != nil {
			return domain.Review{}, err
		}
		if !projection.Diff {
			for index := range files {
				files[index].Patch = ""
			}
		}
	}
	var commits []domain.Commit
	if projection.Commits {
		commits, err = c.reviewCommits(ctx, repository, number)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var comments []reviewComment
	if projection.Threads {
		comments, err = c.reviewComments(ctx, repository, number)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var statuses []domain.BuildStatus
	if projection.Statuses {
		if !isPathSegment(pr.Source.Commit) {
			return domain.Review{}, fmt.Errorf("Data Center pull request source commit is invalid")
		}
		statuses, err = c.reviewStatuses(ctx, pr.Source.Commit)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var participants []domain.Participant
	if projection.Participants {
		participants = mapParticipants(payload.Participants)
	}
	viewerID := ""
	if projection.Viewer {
		viewerID = strings.TrimSpace(c.authentication.Username)
	}
	var threadCount *int
	if projection.ThreadCount && projection.Threads {
		count := len(mapReviewThreads(comments))
		threadCount = &count
	}
	return domain.Review{
		PullRequest:           pr,
		ViewerID:              viewerID,
		Diff:                  diff,
		Files:                 files,
		Commits:               commits,
		Participants:          participants,
		Threads:               mapReviewThreads(comments),
		Statuses:              statuses,
		UnresolvedThreadCount: threadCount,
	}, nil
}

func (c *Client) MutatePullRequest(ctx context.Context, repository domain.Repository, number int, input domain.MutationInput) (domain.PullRequest, error) {
	capability, err := input.Kind.Capability()
	if err != nil {
		return domain.PullRequest{}, err
	}
	if !c.Capabilities().Supports(capability) {
		return domain.PullRequest{}, fmt.Errorf("Data Center does not support %s", capability)
	}
	version := 0
	if input.Kind == domain.MutationMerge || input.Kind == domain.MutationDecline {
		pullRequest, err := c.GetPullRequest(ctx, repository, number)
		if err != nil {
			return domain.PullRequest{}, err
		}
		version = pullRequest.Version
	}
	return c.mutatePullRequest(ctx, repository, number, version, input)
}

func (c *Client) mutatePullRequest(ctx context.Context, repository domain.Repository, number, version int, input domain.MutationInput) (domain.PullRequest, error) {
	capability, err := input.Kind.Capability()
	if err != nil {
		return domain.PullRequest{}, err
	}
	if !c.Capabilities().Supports(capability) {
		return domain.PullRequest{}, fmt.Errorf("Data Center does not support %s", capability)
	}
	if err := validateRepository(repository); err != nil || number <= 0 {
		return domain.PullRequest{}, fmt.Errorf("invalid Data Center pull request")
	}
	endpoint := c.repositoryEndpoint(repository, "pull-requests", fmt.Sprint(number))
	method, body := http.MethodPost, any(nil)
	switch input.Kind {
	case domain.MutationApprove:
		endpoint.Path = path.Join(endpoint.Path, "approve")
	case domain.MutationUnapprove:
		method = http.MethodDelete
		endpoint.Path = path.Join(endpoint.Path, "approve")
	case domain.MutationMerge:
		if version <= 0 {
			return domain.PullRequest{}, fmt.Errorf("Data Center pull request version is required to merge")
		}
		endpoint.Path = path.Join(endpoint.Path, "merge")
		query := endpoint.Query()
		query.Set("version", fmt.Sprint(version))
		endpoint.RawQuery = query.Encode()
	case domain.MutationDecline:
		if version <= 0 {
			return domain.PullRequest{}, fmt.Errorf("Data Center pull request version is required to decline")
		}
		endpoint.Path = path.Join(endpoint.Path, "decline")
		query := endpoint.Query()
		query.Set("version", fmt.Sprint(version))
		endpoint.RawQuery = query.Encode()
	case domain.MutationAddComment, domain.MutationReply:
		if input.Comment == "" {
			return domain.PullRequest{}, fmt.Errorf("Data Center review comment must not be empty")
		}
		endpoint.Path = path.Join(endpoint.Path, "comments")
		body = map[string]any{"text": input.Comment}
		if input.Kind == domain.MutationReply {
			parentID, err := parseCommentID(input.ParentCommentID)
			if err != nil {
				return domain.PullRequest{}, err
			}
			body.(map[string]any)["parent"] = map[string]int{"id": parentID}
		}
	}
	if err := c.json(ctx, method, &endpoint, body, nil); err != nil {
		return domain.PullRequest{}, err
	}
	return c.GetPullRequest(ctx, repository, number)
}
func (c *Client) ApplyReviewAction(ctx context.Context, pullRequest domain.PullRequest, action domain.ReviewAction) (domain.PullRequest, error) {
	return c.MutatePullRequest(ctx, pullRequest.Repository, pullRequest.Number, action)
}
func (c *Client) Health(ctx context.Context) error { _, err := c.Probe(ctx); return err }
func (c *Client) ResolveGitCredential(ctx context.Context) (domain.GitCredential, error) {
	if c.tokenSource == nil {
		return domain.GitCredential{}, fmt.Errorf("Data Center token source is not configured")
	}
	token, expiresAt, err := c.accessToken(ctx)
	if err != nil {
		return domain.GitCredential{}, err
	}
	username, err := c.gitUsername()
	if err != nil {
		return domain.GitCredential{}, err
	}
	return domain.GitCredential{Username: username, Secret: token, ExpiresAt: expiresAt}, nil
}

// ListBranches lists all Data Center branches using start/limit pagination.
func (c *Client) ListBranches(ctx context.Context, repository domain.Repository) ([]domain.Branch, error) {
	return c.listBranches(ctx, repository, "", maxBranchEntries)
}
func (c *Client) listBranches(ctx context.Context, repository domain.Repository, search string, limit int) ([]domain.Branch, error) {
	if err := validateRepository(repository); err != nil {
		return nil, err
	}
	if limit <= 0 {
		return nil, fmt.Errorf("branch limit must be positive")
	}
	endpoint := c.repositoryEndpoint(repository, "branches")
	var branches []domain.Branch
	completed := false
	for start, pages := 0, 0; len(branches) < limit; {
		if pages >= maxBranchPages {
			return nil, fmt.Errorf("Data Center branch pagination limit exceeded")
		}
		query := endpoint.Query()
		query.Set("start", fmt.Sprint(start))
		query.Set("limit", fmt.Sprint(min(limit, maxPageLength)))
		if search != "" {
			query.Set("filterText", search)
		}
		endpoint.RawQuery = query.Encode()
		var page branchPage
		if err := c.getJSON(ctx, &endpoint, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Values {
			if item.DisplayID == "" {
				return nil, fmt.Errorf("Data Center branch omitted a display id")
			}
			branches = append(branches, domain.Branch{Name: item.DisplayID, Commit: item.LatestCommit, IsDefault: item.IsDefault})
			if len(branches) == limit {
				break
			}
		}
		if page.IsLastPage {
			completed = true
			break
		}
		if page.NextPageStart <= start {
			return nil, fmt.Errorf("Data Center branch pagination did not advance")
		}
		start = page.NextPageStart
		pages++
	}
	if !completed {
		return nil, fmt.Errorf("Data Center branch limit exceeded")
	}
	return branches, nil
}

// SearchPullRequests lists Data Center pull requests by title search and requested state.
func (c *Client) SearchPullRequests(ctx context.Context, query domain.PullRequestQuery) ([]domain.PullRequest, error) {
	if !c.Capabilities().Supports(domain.CapabilityPullRequests) {
		return nil, fmt.Errorf("Data Center does not support %s for this authentication mode", domain.CapabilityPullRequests)
	}
	state, err := dataCenterPullRequestState(query.State)
	if err != nil {
		return nil, err
	}
	return c.listPullRequestsForState(ctx, query.Repository, query.Text, state, query.Limit)
}

// SearchPullRequestsPage returns one bounded Data Center page. NextCursor is
// the server-issued nextPageStart value and callers must treat it as opaque.
func (c *Client) SearchPullRequestsPage(ctx context.Context, query domain.PullRequestQuery) (domain.PullRequestPage, error) {
	if !c.Capabilities().Supports(domain.CapabilityPullRequests) {
		return domain.PullRequestPage{}, fmt.Errorf("Data Center does not support %s for this authentication mode", domain.CapabilityPullRequests)
	}
	state, err := dataCenterPullRequestState(query.State)
	if err != nil {
		return domain.PullRequestPage{}, err
	}
	if err := validateRepository(query.Repository); err != nil || query.Limit <= 0 {
		return domain.PullRequestPage{}, fmt.Errorf("invalid Data Center pull request query")
	}
	start := 0
	if query.Cursor != "" {
		start, err = strconv.Atoi(query.Cursor)
		if err != nil || start < 0 {
			return domain.PullRequestPage{}, fmt.Errorf("invalid Data Center pull request cursor")
		}
	}
	endpoint := c.repositoryEndpoint(query.Repository, "pull-requests")
	parameters := endpoint.Query()
	parameters.Set("state", state)
	parameters.Set("start", fmt.Sprint(start))
	parameters.Set("limit", fmt.Sprint(min(query.Limit, maxPageLength)))
	if query.Text != "" {
		parameters.Set("filterText", query.Text)
	}
	endpoint.RawQuery = parameters.Encode()
	var page pullRequestPage
	if err := c.getJSON(ctx, &endpoint, &page); err != nil {
		return domain.PullRequestPage{}, err
	}
	pullRequests := make([]domain.PullRequest, 0, len(page.Values))
	for _, item := range page.Values {
		pullRequest, err := c.mapPullRequest(query.Repository, item)
		if err != nil {
			return domain.PullRequestPage{}, err
		}
		pullRequest.Capabilities = c.Capabilities()
		pullRequests = append(pullRequests, pullRequest)
	}
	nextCursor := ""
	if !page.IsLastPage {
		if page.NextPageStart <= start {
			return domain.PullRequestPage{}, fmt.Errorf("Data Center pull request pagination did not advance")
		}
		nextCursor = fmt.Sprint(page.NextPageStart)
	}
	return domain.PullRequestPage{PullRequests: pullRequests, NextCursor: nextCursor}, nil
}

func (c *Client) listPullRequests(ctx context.Context, repository domain.Repository, search string, limit int) ([]domain.PullRequest, error) {
	return c.listPullRequestsForState(ctx, repository, search, "OPEN", limit)
}

func (c *Client) listPullRequestsForState(ctx context.Context, repository domain.Repository, search, state string, limit int) ([]domain.PullRequest, error) {
	if err := validateRepository(repository); err != nil {
		return nil, err
	}
	if limit <= 0 {
		return nil, fmt.Errorf("pull request limit must be positive")
	}
	endpoint := c.repositoryEndpoint(repository, "pull-requests")
	var pullRequests []domain.PullRequest
	for start := 0; len(pullRequests) < limit; {
		query := endpoint.Query()
		query.Set("state", state)
		query.Set("start", fmt.Sprint(start))
		query.Set("limit", fmt.Sprint(min(limit, maxPageLength)))
		if search != "" {
			query.Set("filterText", search)
		}
		endpoint.RawQuery = query.Encode()
		var page pullRequestPage
		if err := c.getJSON(ctx, &endpoint, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Values {
			pullRequest, err := c.mapPullRequest(repository, item)
			if err != nil {
				return nil, err
			}
			pullRequest.Capabilities = c.Capabilities()
			pullRequests = append(pullRequests, pullRequest)
			if len(pullRequests) == limit {
				break
			}
		}
		if page.IsLastPage {
			break
		}
		if page.NextPageStart <= start {
			return nil, fmt.Errorf("Data Center pull request pagination did not advance")
		}
		start = page.NextPageStart
	}
	return pullRequests, nil
}

func dataCenterPullRequestState(raw string) (string, error) {
	switch state := strings.ToUpper(strings.TrimSpace(raw)); state {
	case "", "OPEN":
		return "OPEN", nil
	case "MERGED", "DECLINED", "ALL":
		return state, nil
	default:
		return "", fmt.Errorf("unsupported Data Center pull request state")
	}
}
