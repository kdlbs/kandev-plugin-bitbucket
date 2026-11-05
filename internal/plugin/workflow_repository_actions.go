package plugin

import (
	"context"
	"fmt"
	"net/url"

	"kandev-plugin-bitbucket/internal/domain"

	"github.com/kdlbs/kandev/pluginsdk"
)

func (w *Workflows) handleRepositoryAction(ctx context.Context, request *pluginsdk.PluginActionRequest) (*pluginsdk.PluginActionResponse, error) {
	switch request.ActionKey {
	case "repositories.list":
		var input listRepositoriesInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		unconfigured, err := w.workspaceIsUnconfigured(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		if unconfigured {
			return actionResponse(map[string]any{"repositories": []any{}})
		}
		provider, err := w.provider(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		page, err := listRepositoryPage(ctx, provider, domain.RepositoryQuery{
			Text: input.Query, Limit: boundedLimit(input.Limit), Cursor: input.Cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("list repositories: %w", err)
		}
		return actionResponse(map[string]any{
			"repositories": repositoryViews(page.Repositories), "next_cursor": page.NextCursor,
		})
	case "repositories.branches":
		var input repositoryInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		provider, repository, err := w.repository(ctx, request.Context.WorkspaceID, input.Repository)
		if err != nil {
			return nil, err
		}
		if err := requireCapability(provider, domain.CapabilityBranches); err != nil {
			return nil, err
		}
		branches, err := provider.ListBranches(ctx, repository)
		if err != nil {
			return nil, fmt.Errorf("list branches: %w", err)
		}
		return actionResponse(map[string]any{"branches": branchViews(branches)})
	case "repositories.inspect":
		var input repositoryInspectInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		parsed, err := url.Parse(input.URL)
		if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Scheme != "https" {
			return nil, invalidActionError("repository URL must be a credential-free HTTPS URL")
		}
		unconfigured, err := w.workspaceIsUnconfigured(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		if unconfigured {
			return actionResponse(map[string]any{"matched": false})
		}
		provider, err := w.provider(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		repository, err := provider.InspectRepositoryURL(input.URL)
		if err == nil {
			repository, err = hydrateRepositoryIdentity(ctx, provider, repository)
			if err != nil {
				return nil, notFoundActionError("Bitbucket repository is unavailable")
			}
			return actionResponse(repositoryViews([]domain.Repository{repository})[0])
		}
		locator, inspectErr := provider.InspectPullRequestURL(input.URL)
		if inspectErr != nil {
			return actionResponse(map[string]any{"matched": false})
		}
		locator.Repository, inspectErr = hydrateRepositoryIdentity(ctx, provider, locator.Repository)
		if inspectErr != nil {
			return nil, notFoundActionError("Bitbucket repository is unavailable")
		}
		pullRequest, getErr := provider.GetPullRequest(ctx, locator.Repository, locator.Number)
		if getErr != nil {
			return nil, notFoundActionError("Bitbucket pull request is unavailable")
		}
		repositoryView := repositoryViews([]domain.Repository{pullRequest.Repository})[0]
		repositoryView["base_branch"] = pullRequest.Destination.Name
		repositoryView["head_branch"] = pullRequest.Source.Name
		return actionResponse(map[string]any{
			"repository": repositoryView, "pull_request": pullRequestView(pullRequest),
			"base_branch": pullRequest.Destination.Name, "head_branch": pullRequest.Source.Name,
		})
	default:
		return nil, notFoundActionError("unsupported Bitbucket action %q", request.ActionKey)
	}
}
