package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"
)

func (c *Client) Capabilities() domain.Capabilities {
	return domain.Capabilities{domain.CapabilityBranches: true, domain.CapabilityPullRequests: true, domain.CapabilityReview: true, domain.CapabilityApprove: true, domain.CapabilityMerge: true, domain.CapabilityDecline: true, domain.CapabilityComments: true, domain.CapabilityThreadReplies: true, domain.CapabilityBuildStatuses: true, domain.CapabilityBuildActions: false, domain.CapabilityIssues: false}
}

func (c *Client) GetPullRequest(ctx context.Context, repository domain.Repository, number int) (domain.PullRequest, error) {
	if err := validateRepository(repository); err != nil || number <= 0 {
		return domain.PullRequest{}, fmt.Errorf("invalid Cloud pull request")
	}
	endpoint := c.repositoryEndpoint(repository, "pullrequests", fmt.Sprint(number))
	var payload pullRequestPayload
	if err := c.getJSON(ctx, &endpoint, &payload); err != nil {
		return domain.PullRequest{}, err
	}
	pullRequest, err := mapPullRequest(repository, payload)
	pullRequest.Capabilities = c.Capabilities()
	return pullRequest, err
}

func (c *Client) CreatePullRequest(ctx context.Context, input domain.CreatePullRequestInput) (domain.PullRequest, error) {
	if err := validateRepository(input.Repository); err != nil || input.Title == "" || input.Source == "" || input.Destination == "" {
		return domain.PullRequest{}, fmt.Errorf("invalid Cloud pull request input")
	}
	endpoint := c.repositoryEndpoint(input.Repository, "pullrequests")
	var payload pullRequestPayload
	body := map[string]any{"title": input.Title, "description": input.Description, "source": map[string]any{"branch": map[string]string{"name": input.Source}}, "destination": map[string]any{"branch": map[string]string{"name": input.Destination}}, "close_source_branch": input.CloseSourceOnMerge}
	if err := c.json(ctx, http.MethodPost, &endpoint, body, &payload); err != nil {
		return domain.PullRequest{}, err
	}
	pullRequest, err := mapPullRequest(input.Repository, payload)
	pullRequest.Capabilities = c.Capabilities()
	return pullRequest, err
}

func (c *Client) GetReview(ctx context.Context, repository domain.Repository, number int) (domain.Review, error) {
	return c.GetReviewProjected(ctx, repository, number, domain.FullReviewProjection())
}

func (c *Client) GetReviewProjected(ctx context.Context, repository domain.Repository, number int, projection domain.ReviewProjection) (domain.Review, error) {
	if err := validateRepository(repository); err != nil || number <= 0 {
		return domain.Review{}, fmt.Errorf("invalid Cloud pull request")
	}
	pullRequestEndpoint := c.repositoryEndpoint(repository, "pullrequests", fmt.Sprint(number))
	var payload pullRequestPayload
	if err := c.getJSON(ctx, &pullRequestEndpoint, &payload); err != nil {
		return domain.Review{}, err
	}
	pr, err := mapPullRequest(repository, payload)
	if err != nil {
		return domain.Review{}, err
	}
	pr.Capabilities = c.Capabilities()
	diff := ""
	if projection.Diff {
		diffEndpoint := c.repositoryEndpoint(repository, "pullrequests", fmt.Sprint(number), "diff")
		diff, err = c.getText(ctx, &diffEndpoint)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var files []domain.ReviewFile
	if projection.Files {
		files, err = c.reviewFiles(ctx, repository, number, diff)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var commits []domain.Commit
	if projection.Commits {
		commits, err = c.reviewCommits(ctx, repository, number)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var comments []domain.Thread
	if projection.Threads {
		comments, err = c.reviewComments(ctx, repository, number)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var statuses []domain.BuildStatus
	if projection.Statuses {
		if !isPathSegment(pr.Source.Commit) {
			return domain.Review{}, fmt.Errorf("Cloud pull request source commit is invalid")
		}
		statuses, err = c.reviewStatuses(ctx, repository, pr.Source.Commit)
		if err != nil {
			return domain.Review{}, err
		}
	}
	viewerID := ""
	if projection.Viewer {
		viewerID, err = c.currentUserID(ctx)
		if err != nil {
			return domain.Review{}, err
		}
	}
	var threadCount *int
	if projection.ThreadCount && projection.Threads {
		count := len(comments)
		threadCount = &count
	}
	var participants []domain.Participant
	if projection.Participants {
		participants = mapParticipants(payload.Participants)
	}
	return domain.Review{
		PullRequest:           pr,
		ViewerID:              viewerID,
		Diff:                  diff,
		Files:                 files,
		Commits:               commits,
		Participants:          participants,
		Threads:               comments,
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
		return domain.PullRequest{}, fmt.Errorf("Cloud does not support %s", capability)
	}
	if err := validateRepository(repository); err != nil || number <= 0 {
		return domain.PullRequest{}, fmt.Errorf("invalid Cloud pull request")
	}
	endpoint := c.repositoryEndpoint(repository, "pullrequests", fmt.Sprint(number))
	method, body := http.MethodPost, any(nil)
	switch input.Kind {
	case domain.MutationApprove:
		endpoint.Path = path.Join(endpoint.Path, "approve")
	case domain.MutationUnapprove:
		method = http.MethodDelete
		endpoint.Path = path.Join(endpoint.Path, "approve")
	case domain.MutationMerge:
		endpoint.Path = path.Join(endpoint.Path, "merge")
	case domain.MutationDecline:
		endpoint.Path = path.Join(endpoint.Path, "decline")
	case domain.MutationAddComment, domain.MutationReply:
		if input.Comment == "" {
			return domain.PullRequest{}, fmt.Errorf("Cloud review comment must not be empty")
		}
		endpoint.Path = path.Join(endpoint.Path, "comments")
		body = map[string]any{"content": map[string]string{"raw": input.Comment}}
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
func (c *Client) Health(ctx context.Context) error {
	endpoint := *c.apiBase
	endpoint.Path = path.Join(endpoint.Path, "user")
	var value map[string]any
	return c.getJSON(ctx, &endpoint, &value)
}

func (c *Client) currentUserID(ctx context.Context) (string, error) {
	endpoint := *c.apiBase
	endpoint.Path = path.Join(endpoint.Path, "user")
	var value struct {
		AccountID string `json:"account_id"`
	}
	if err := c.getJSON(ctx, &endpoint, &value); err != nil {
		return "", err
	}
	if strings.TrimSpace(value.AccountID) == "" {
		return "", fmt.Errorf("Cloud current user omitted an account id")
	}
	return value.AccountID, nil
}
func (c *Client) ResolveGitCredential(ctx context.Context) (domain.GitCredential, error) {
	if c.tokenSource == nil {
		return domain.GitCredential{}, fmt.Errorf("Cloud token source is not configured")
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

// ListBranches lists all branches from the Cloud v2 branch endpoint.
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
	endpoint := c.repositoryEndpoint(repository, "refs", "branches")
	query := endpoint.Query()
	query.Set("pagelen", fmt.Sprint(min(limit, maxPageLength)))
	if search != "" {
		query.Set("q", fmt.Sprintf("name~%q", search))
	}
	endpoint.RawQuery = query.Encode()

	var branches []domain.Branch
	pages := 0
	next := &endpoint
	for next != nil && len(branches) < limit {
		if pages >= maxBranchPages {
			return nil, fmt.Errorf("Cloud branch pagination limit exceeded")
		}
		var page branchPage
		if err := c.getJSON(ctx, next, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Values {
			if item.Name == "" {
				return nil, fmt.Errorf("Cloud branch omitted a name")
			}
			branches = append(branches, domain.Branch{Name: item.Name, Commit: item.Target.Hash})
			if len(branches) == limit {
				break
			}
		}
		var err error
		next, err = c.nextURL(next, page.Next)
		if err != nil {
			return nil, err
		}
		pages++
	}
	if next != nil {
		return nil, fmt.Errorf("Cloud branch limit exceeded")
	}
	return branches, nil
}

// SearchPullRequests lists pull requests matching a title search and requested state.
func (c *Client) SearchPullRequests(ctx context.Context, query domain.PullRequestQuery) ([]domain.PullRequest, error) {
	if query.Limit <= 0 {
		return nil, fmt.Errorf("pull request limit must be positive")
	}
	states, err := cloudPullRequestStates(query.State)
	if err != nil {
		return nil, err
	}
	pullRequests := make([]domain.PullRequest, 0, query.Limit)
	seen := make(map[string]struct{})
	for _, state := range states {
		results, searchErr := c.listPullRequestsForState(ctx, query.Repository, query.Text, state, query.Limit)
		if searchErr != nil {
			return nil, searchErr
		}
		for _, pullRequest := range results {
			if _, found := seen[pullRequest.Key()]; found {
				continue
			}
			seen[pullRequest.Key()] = struct{}{}
			pullRequests = append(pullRequests, pullRequest)
		}
	}
	sort.Slice(pullRequests, func(i, j int) bool { return pullRequests[i].Key() < pullRequests[j].Key() })
	if len(pullRequests) > query.Limit {
		return pullRequests[:query.Limit], nil
	}
	return pullRequests, nil
}

// SearchPullRequestsPage returns one bounded Cloud page. NextCursor is an
// opaque, origin-validated API URL and is safe to persist only for this query.
func (c *Client) SearchPullRequestsPage(ctx context.Context, query domain.PullRequestQuery) (domain.PullRequestPage, error) {
	if query.Limit <= 0 {
		return domain.PullRequestPage{}, fmt.Errorf("pull request limit must be positive")
	}
	if query.Repository.ID == "" || query.Repository.ProviderScope == "" {
		return domain.PullRequestPage{}, fmt.Errorf("pull request pagination requires immutable repository identity")
	}
	states, err := cloudPullRequestStates(query.State)
	if err != nil {
		return domain.PullRequestPage{}, err
	}
	if len(states) > 1 {
		return c.searchAllPullRequestsPage(ctx, query, states)
	}
	return c.searchPullRequestsSinglePage(ctx, query, states[0])
}

const cloudAllPageCursorVersion = 2

type cloudAllPageCursor struct {
	Version       int    `json:"version"`
	ProviderScope string `json:"provider_scope"`
	RepositoryID  string `json:"repository_id"`
	QueryHash     string `json:"query_hash"`
	State         int    `json:"state"`
	Cursor        string `json:"cursor,omitempty"`
}

func (c *Client) searchAllPullRequestsPage(ctx context.Context, query domain.PullRequestQuery, states []string) (domain.PullRequestPage, error) {
	cursor, err := parseCloudAllPageCursor(query.Cursor, query)
	if err != nil {
		return domain.PullRequestPage{}, err
	}
	if cursor.State >= len(states) {
		return domain.PullRequestPage{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	pageQuery := query
	pageQuery.Cursor = cursor.Cursor
	page, err := c.searchPullRequestsSinglePage(ctx, pageQuery, states[cursor.State])
	if err != nil {
		return domain.PullRequestPage{}, err
	}
	if page.NextCursor != "" {
		page.NextCursor = encodeCloudAllPageCursor(newCloudAllPageCursor(query, cursor.State, page.NextCursor))
		return page, nil
	}
	if cursor.State+1 < len(states) {
		page.NextCursor = encodeCloudAllPageCursor(newCloudAllPageCursor(query, cursor.State+1, ""))
	}
	return page, nil
}

func (c *Client) searchPullRequestsSinglePage(ctx context.Context, query domain.PullRequestQuery, state string) (domain.PullRequestPage, error) {
	endpoint, err := c.pullRequestPageEndpoint(query, state)
	if err != nil {
		return domain.PullRequestPage{}, err
	}
	var page pullRequestPage
	if err := c.getJSON(ctx, endpoint, &page); err != nil {
		return domain.PullRequestPage{}, err
	}
	pullRequests := make([]domain.PullRequest, 0, len(page.Values))
	for _, item := range page.Values {
		pullRequest, err := mapPullRequest(query.Repository, item)
		if err != nil {
			return domain.PullRequestPage{}, err
		}
		pullRequest.Capabilities = c.Capabilities()
		pullRequests = append(pullRequests, pullRequest)
	}
	next, err := c.nextURL(endpoint, page.Next)
	if err != nil {
		return domain.PullRequestPage{}, err
	}
	nextCursor := ""
	if next != nil {
		nextCursor = encodeCloudPullRequestCursor(newCloudPullRequestCursor(query, state, next.String()))
	}
	return domain.PullRequestPage{PullRequests: pullRequests, NextCursor: nextCursor}, nil
}

func encodeCloudAllPageCursor(cursor cloudAllPageCursor) string {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func newCloudAllPageCursor(query domain.PullRequestQuery, state int, cursor string) cloudAllPageCursor {
	return cloudAllPageCursor{
		Version: cloudAllPageCursorVersion, ProviderScope: query.Repository.ProviderScope,
		RepositoryID: query.Repository.ID, QueryHash: cloudPullRequestQueryHash(query, "ALL"),
		State: state, Cursor: cursor,
	}
}

func parseCloudAllPageCursor(raw string, query domain.PullRequestQuery) (cloudAllPageCursor, error) {
	if raw == "" {
		return newCloudAllPageCursor(query, 0, ""), nil
	}
	if len(raw) > 8192 {
		return cloudAllPageCursor{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cloudAllPageCursor{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	var cursor cloudAllPageCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.Version != cloudAllPageCursorVersion || cursor.State < 0 ||
		cursor.ProviderScope != query.Repository.ProviderScope || cursor.RepositoryID != query.Repository.ID ||
		cursor.QueryHash != cloudPullRequestQueryHash(query, "ALL") {
		return cloudAllPageCursor{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	return cursor, nil
}

func (c *Client) pullRequestPageEndpoint(query domain.PullRequestQuery, state string) (*url.URL, error) {
	if err := validateRepository(query.Repository); err != nil {
		return nil, err
	}
	endpoint := c.repositoryEndpoint(query.Repository, "pullrequests")
	if query.Cursor != "" {
		cursor, err := parseCloudPullRequestCursor(query.Cursor, query, state)
		if err != nil {
			return nil, err
		}
		next, err := c.nextURL(&endpoint, cursor.NextURL)
		if err != nil || next == nil || !validPullRequestPageURL(next, endpoint.Path, query, state) {
			return nil, fmt.Errorf("invalid Cloud pull request cursor")
		}
		return next, nil
	}
	parameters := endpoint.Query()
	parameters.Set("state", state)
	parameters.Set("pagelen", fmt.Sprint(min(query.Limit, maxPullRequestPageLength)))
	search := strings.TrimSpace(query.Text)
	if search != "" {
		parameters.Set("q", fmt.Sprintf("title~%q", search))
	}
	endpoint.RawQuery = parameters.Encode()
	return &endpoint, nil
}

const cloudPullRequestCursorVersion = 1

type cloudPullRequestCursor struct {
	Version       int    `json:"version"`
	ProviderScope string `json:"provider_scope"`
	RepositoryID  string `json:"repository_id"`
	QueryHash     string `json:"query_hash"`
	NextURL       string `json:"next_url"`
}

func newCloudPullRequestCursor(query domain.PullRequestQuery, state, nextURL string) cloudPullRequestCursor {
	return cloudPullRequestCursor{
		Version: cloudPullRequestCursorVersion, ProviderScope: query.Repository.ProviderScope,
		RepositoryID: query.Repository.ID, QueryHash: cloudPullRequestQueryHash(query, state), NextURL: nextURL,
	}
}

func encodeCloudPullRequestCursor(cursor cloudPullRequestCursor) string {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func parseCloudPullRequestCursor(raw string, query domain.PullRequestQuery, state string) (cloudPullRequestCursor, error) {
	if raw == "" || len(raw) > 8192 || query.Repository.ID == "" || query.Repository.ProviderScope == "" {
		return cloudPullRequestCursor{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cloudPullRequestCursor{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	var cursor cloudPullRequestCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.Version != cloudPullRequestCursorVersion || cursor.NextURL == "" ||
		cursor.ProviderScope != query.Repository.ProviderScope || cursor.RepositoryID != query.Repository.ID ||
		cursor.QueryHash != cloudPullRequestQueryHash(query, state) {
		return cloudPullRequestCursor{}, fmt.Errorf("invalid Cloud pull request cursor")
	}
	return cursor, nil
}

func cloudPullRequestQueryHash(query domain.PullRequestQuery, state string) string {
	payload := query.Repository.ProviderScope + "\x00" + query.Repository.ID + "\x00" +
		strings.TrimSpace(query.Text) + "\x00" + strings.ToUpper(strings.TrimSpace(state)) + "\x00" + fmt.Sprint(query.Limit)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
}

func validPullRequestPageURL(next *url.URL, expectedPath string, query domain.PullRequestQuery, state string) bool {
	if next == nil || next.Path != expectedPath {
		return false
	}
	values := next.Query()
	if values.Get("state") != state || values.Get("pagelen") != fmt.Sprint(min(query.Limit, maxPullRequestPageLength)) {
		return false
	}
	expectedQuery := ""
	if search := strings.TrimSpace(query.Text); search != "" {
		expectedQuery = fmt.Sprintf("title~%q", search)
	}
	return values.Get("q") == expectedQuery
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
	endpoint := c.repositoryEndpoint(repository, "pullrequests")
	query := endpoint.Query()
	query.Set("state", state)
	query.Set("pagelen", fmt.Sprint(min(limit, maxPullRequestPageLength)))
	if search != "" {
		query.Set("q", fmt.Sprintf("title~%q", search))
	}
	endpoint.RawQuery = query.Encode()

	var pullRequests []domain.PullRequest
	for next := &endpoint; next != nil && len(pullRequests) < limit; {
		var page pullRequestPage
		if err := c.getJSON(ctx, next, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Values {
			pullRequest, err := mapPullRequest(repository, item)
			if err != nil {
				return nil, err
			}
			pullRequest.Capabilities = c.Capabilities()
			pullRequests = append(pullRequests, pullRequest)
			if len(pullRequests) == limit {
				break
			}
		}
		var err error
		next, err = c.nextURL(next, page.Next)
		if err != nil {
			return nil, err
		}
	}
	return pullRequests, nil
}

func cloudPullRequestStates(raw string) ([]string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "OPEN":
		return []string{"OPEN"}, nil
	case "MERGED", "DECLINED", "SUPERSEDED":
		return []string{strings.ToUpper(strings.TrimSpace(raw))}, nil
	case "ALL":
		return []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"}, nil
	default:
		return nil, fmt.Errorf("unsupported Cloud pull request state")
	}
}
