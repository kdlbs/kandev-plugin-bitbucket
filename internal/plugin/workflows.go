package plugin

import (
	"context"
	"fmt"
	"net/url"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kandev/kandev/pkg/pluginsdk"
)

const referenceSource = "bitbucket"

// Workflows implements every authenticated plugin action plus composer and
// credential RPCs. Its inputs are host-verified action context and adapter
// DTOs; it never receives a repository URL with embedded credentials.
type Workflows struct {
	host        pluginsdk.Host
	resolver    ProviderResolver
	tasks       *TaskGateway
	links       *LinkStore
	watches     *watches.Service
	credentials *CredentialResolver
}

func NewWorkflows(host pluginsdk.Host, resolver ProviderResolver) (*Workflows, error) {
	if host == nil || resolver == nil {
		return nil, fmt.Errorf("host and provider resolver are required")
	}
	state, err := NewWatchStateRepository(host)
	if err != nil {
		return nil, err
	}
	taskGateway, err := NewTaskGateway(host)
	if err != nil {
		return nil, err
	}
	links, err := NewLinkStore(host)
	if err != nil {
		return nil, err
	}
	events, err := NewEventSink(host)
	if err != nil {
		return nil, err
	}
	watchProvider, err := NewWatchProvider(resolver)
	if err != nil {
		return nil, err
	}
	watchService, err := watches.NewService(watches.Options{Repository: state, Provider: watchProvider, Tasks: taskGateway, Events: events})
	if err != nil {
		return nil, err
	}
	credentials, err := NewCredentialResolver(providerCredentialSource{resolver: resolver})
	if err != nil {
		return nil, err
	}
	return &Workflows{host: host, resolver: resolver, tasks: taskGateway, links: links, watches: watchService, credentials: credentials}, nil
}

func (w *Workflows) HandleAction(ctx context.Context, request *pluginsdk.PluginActionRequest) (*pluginsdk.PluginActionResponse, error) {
	if request == nil {
		return nil, invalidActionError("action request is required")
	}
	if request.Context.WorkspaceID == "" {
		return nil, forbiddenActionError("verified workspace context is required")
	}
	switch request.ActionKey {
	case "source.read", "source.create":
		return w.handleSourceAction(ctx, request)
	case "connection.get", "connection.disconnect", "connection.save", "oauth.start":
		return w.handleConnectionAction(ctx, request)
	case "repositories.list", "repositories.branches", "repositories.inspect":
		return w.handleRepositoryAction(ctx, request)
	case "pullrequests.search", "pullrequests.queue", "pullrequests.associations",
		"pullrequests.get", "pullrequests.inspect", "pullrequests.create",
		"reviews.get", "reviews.action", "tasks.launch",
		"pullrequests.link", "pullrequests.unlink":
		return w.handlePullRequestAction(ctx, request)
	case "watches.get", "watches.create", "watches.update",
		"watches.filter", "watches.preset", "watches.run",
		"watches.pause", "watches.resume",
		"watches.preview_reset", "watches.preview_delete",
		"watches.reset", "watches.delete":
		response, err := w.handleWatchAction(ctx, request)
		return response, categorizeWatchActionError(err)
	default:
		return nil, notFoundActionError("unsupported Bitbucket action %q", request.ActionKey)
	}
}

// HandleWebhook dispatches only the declared OAuth callback and keeps all
// callback parameters inside the PKCE coordinator rather than action bodies.
func (w *Workflows) HandleWebhook(ctx context.Context, request *pluginsdk.WebhookRequest) (*pluginsdk.WebhookResponse, error) {
	if request == nil || request.WebhookKey != "oauth-callback" || request.Method != "GET" {
		return &pluginsdk.WebhookResponse{Status: 404}, nil
	}
	callback, ok := w.resolver.(interface {
		HandleOAuthCallback(context.Context, string, string) (*pluginsdk.WebhookResponse, error)
	})
	if !ok {
		return &pluginsdk.WebhookResponse{Status: 404}, nil
	}
	query, err := url.ParseQuery(request.Query)
	if err != nil {
		return &pluginsdk.WebhookResponse{Status: 400}, nil
	}
	response, err := callback.HandleOAuthCallback(ctx, query.Get("state"), query.Get("code"))
	if err != nil {
		return &pluginsdk.WebhookResponse{Status: 400, Headers: map[string]string{"Content-Type": "text/plain; charset=utf-8"}, Body: []byte("OAuth callback could not be completed.")}, nil
	}
	return response, nil
}

func (w *Workflows) SearchEntityReferences(ctx context.Context, request *pluginsdk.SearchEntityReferencesRequest) (*pluginsdk.SearchEntityReferencesResponse, error) {
	if request == nil || request.Source != referenceSource || request.WorkspaceID == "" {
		return nil, invalidActionError("invalid Bitbucket reference search")
	}
	provider, err := w.provider(ctx, request.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(provider, domain.CapabilityPullRequests); err != nil {
		return nil, err
	}
	// Candidate count must not truncate repository discovery: a matching pull
	// request may live after repositories that return no matches.
	repositories, err := listAllRepositories(ctx, provider, "")
	if err != nil {
		return nil, fmt.Errorf("list reference repositories: %w", err)
	}
	response := &pluginsdk.SearchEntityReferencesResponse{}
	limit := boundedLimit(int(request.Limit))
	for _, repository := range repositories {
		pullRequests, err := provider.SearchPullRequests(ctx, domain.PullRequestQuery{Repository: repository, Text: request.Query, Limit: boundedLimit(int(request.Limit))})
		if err != nil {
			return nil, fmt.Errorf("search Bitbucket references: %w", err)
		}
		for _, pullRequest := range pullRequests {
			response.Candidates = append(response.Candidates, pluginsdk.EntityReferenceCandidate{ProviderLocalID: pullRequest.Key(), Title: pullRequest.Title, URL: pullRequest.URL, Attributes: map[string]any{"key": pullRequest.Key(), "repository": map[string]any{"namespace": pullRequest.Repository.Namespace, "slug": pullRequest.Repository.Slug}, "number": pullRequest.Number}})
			if len(response.Candidates) >= limit {
				return response, nil
			}
		}
	}
	return response, nil
}

func (w *Workflows) AuthorizeEntityReference(ctx context.Context, request *pluginsdk.AuthorizeEntityReferenceRequest) (*pluginsdk.AuthorizeEntityReferenceResponse, error) {
	if request == nil || request.Source != referenceSource || request.WorkspaceID == "" || (request.Purpose != "search" && request.Purpose != "submission") {
		return &pluginsdk.AuthorizeEntityReferenceResponse{Allowed: false, Reason: "invalid Bitbucket reference"}, nil
	}
	repository, number, key, ok := referenceIdentity(request.Reference)
	if !ok {
		return &pluginsdk.AuthorizeEntityReferenceResponse{Allowed: false, Reason: "incomplete Bitbucket reference"}, nil
	}
	provider, err := w.provider(ctx, request.WorkspaceID)
	if err != nil || requireCapability(provider, domain.CapabilityPullRequests) != nil {
		return &pluginsdk.AuthorizeEntityReferenceResponse{Allowed: false, Reason: "Bitbucket connection unavailable"}, nil
	}
	pullRequest, err := provider.GetPullRequest(ctx, repository, number)
	if err != nil || pullRequest.Key() != key {
		return &pluginsdk.AuthorizeEntityReferenceResponse{Allowed: false, Reason: "Bitbucket pull request is unavailable"}, nil
	}
	return &pluginsdk.AuthorizeEntityReferenceResponse{Allowed: true}, nil
}

func (w *Workflows) ResolveGitCredential(ctx context.Context, request *pluginsdk.ResolveGitCredentialRequest) (*pluginsdk.ResolveGitCredentialResponse, error) {
	return w.credentials.ResolveGitCredential(ctx, request)
}

func (w *Workflows) GetGitCredentialBinding(ctx context.Context, request *pluginsdk.GitCredentialBindingRequest) (*pluginsdk.GitCredentialBindingResponse, error) {
	return w.credentials.GetGitCredentialBinding(ctx, request)
}

func (w *Workflows) provider(ctx context.Context, workspaceID string) (domain.Provider, error) {
	if workspaceID == "" {
		return nil, forbiddenActionError("verified workspace context is required")
	}
	provider, err := w.resolver.Provider(ctx, workspaceID)
	if err != nil {
		return nil, unavailableActionError("resolve Bitbucket connection: %v", err)
	}
	return provider, nil
}

func (w *Workflows) workspaceIsUnconfigured(ctx context.Context, workspaceID string) (bool, error) {
	if workspaceID == "" {
		return false, forbiddenActionError("verified workspace context is required")
	}
	connections, ok := w.resolver.(ConnectionSettingsStore)
	if !ok {
		return false, nil
	}
	_, found, err := connections.Load(ctx, workspaceID)
	if err != nil {
		return false, unavailableActionError("load Bitbucket connection: %v", err)
	}
	return !found, nil
}

func (w *Workflows) repository(ctx context.Context, workspaceID string, remote watches.RemoteRepository) (domain.Provider, domain.Repository, error) {
	provider, err := w.provider(ctx, workspaceID)
	if err != nil {
		return nil, domain.Repository{}, err
	}
	repository, err := domainRepository(remote)
	if err != nil {
		return nil, domain.Repository{}, invalidActionError("invalid repository descriptor: %v", err)
	}
	return provider, repository, nil
}

func (w *Workflows) pullRequest(ctx context.Context, workspaceID string, body []byte) (domain.Provider, domain.PullRequest, error) {
	var input pullRequestLookup
	if err := decodeAction(body, &input); err != nil {
		return nil, domain.PullRequest{}, err
	}
	return w.pullRequestLookup(ctx, workspaceID, input)
}

func (w *Workflows) pullRequestLookup(ctx context.Context, workspaceID string, input pullRequestLookup) (domain.Provider, domain.PullRequest, error) {
	if input.PullRequestID != "" && input.Number > 0 && input.PullRequestID != fmt.Sprint(input.Number) {
		return nil, domain.PullRequest{}, invalidActionError("pull request identity is inconsistent")
	}
	immutableFields := 0
	if input.ProviderScope != "" {
		immutableFields++
	}
	if input.RepositoryID != "" {
		immutableFields++
	}
	if input.Number > 0 {
		immutableFields++
	}
	if immutableFields > 0 {
		if immutableFields != 3 {
			return nil, domain.PullRequest{}, invalidActionError("complete pull request identity is required")
		}
		provider, err := w.provider(ctx, workspaceID)
		if err != nil {
			return nil, domain.PullRequest{}, err
		}
		repository, err := persistedRepository(ctx, provider, input.RepositoryID, input.ProviderScope)
		if err != nil {
			return nil, domain.PullRequest{}, notFoundActionError("Bitbucket repository is unavailable")
		}
		pullRequest, err := provider.GetPullRequest(ctx, repository, input.Number)
		if err != nil || pullRequest.Repository.ID != input.RepositoryID ||
			pullRequest.Repository.ProviderScope != input.ProviderScope || pullRequest.Number != input.Number {
			return nil, domain.PullRequest{}, notFoundActionError("Bitbucket pull request is unavailable")
		}
		return provider, pullRequest, nil
	}
	if input.ReviewKey != "" {
		provider, err := w.provider(ctx, workspaceID)
		if err != nil {
			return nil, domain.PullRequest{}, err
		}
		repository, number, key, ok := pullRequestIdentity(provider, input.ReviewKey)
		if !ok {
			return nil, domain.PullRequest{}, invalidActionError("invalid Bitbucket pull request key")
		}
		repository, err = hydrateRepositoryIdentity(ctx, provider, repository)
		if err != nil {
			return nil, domain.PullRequest{}, notFoundActionError("Bitbucket repository is unavailable")
		}
		pullRequest, err := provider.GetPullRequest(ctx, repository, number)
		if err != nil || pullRequest.Key() != key ||
			(input.PullRequestID != "" && input.PullRequestID != fmt.Sprint(pullRequest.Number)) {
			return nil, domain.PullRequest{}, notFoundActionError("Bitbucket pull request is unavailable")
		}
		return provider, pullRequest, nil
	}
	provider, repository, err := w.repository(ctx, workspaceID, input.Repository)
	if err != nil {
		return nil, domain.PullRequest{}, err
	}
	if input.Number <= 0 {
		return nil, domain.PullRequest{}, invalidActionError("pull request number is required")
	}
	pullRequest, err := provider.GetPullRequest(ctx, repository, input.Number)
	if err != nil {
		return nil, domain.PullRequest{}, fmt.Errorf("get pull request: %w", err)
	}
	return provider, pullRequest, nil
}

func pullRequestIdentity(provider domain.Provider, value string) (domain.Repository, int, string, bool) {
	if repository, number, ok := parsePullRequestKey(value); ok {
		return repository, number, value, true
	}
	locator, err := provider.InspectPullRequestURL(value)
	if err != nil || locator.Repository.Namespace == "" || locator.Repository.Slug == "" || locator.Number <= 0 {
		return domain.Repository{}, 0, "", false
	}
	return locator.Repository, locator.Number, fmt.Sprintf("%s/%s#%d", locator.Repository.Namespace, locator.Repository.Slug, locator.Number), true
}
