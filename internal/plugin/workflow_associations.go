package plugin

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

func (w *Workflows) taskPullRequests(ctx context.Context, workspaceID, taskID string) (*pluginsdk.PluginActionResponse, error) {
	links, err := w.links.List(ctx, taskID)
	if err != nil {
		return nil, err
	}
	watchAssociations, err := w.watchOwnedAssociations(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	provider, err := w.provider(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	_ = w.autoLinkTaskPullRequests(ctx, workspaceID, taskID, provider)
	if refreshedLinks, refreshErr := w.links.List(ctx, taskID); refreshErr == nil {
		links = refreshedLinks
	}
	pullRequests := make([]domain.PullRequest, 0, len(links)+len(watchAssociations[taskID]))
	unavailable := make([]map[string]any, 0)
	seen := make(map[string]struct{}, len(links)+len(watchAssociations[taskID]))
	for _, link := range links {
		available, err := w.linkMatchesConnection(ctx, workspaceID, link)
		if err != nil {
			return nil, err
		}
		if !available {
			if !hasWatchAssociationKey(watchAssociations[taskID], link.Key) {
				unavailable = append(unavailable, map[string]any{"key": link.Key, "reason": "connection_changed"})
			}
			continue
		}
		repository, number, ok := parsePullRequestKey(link.Key)
		if !ok {
			continue
		}
		repository, err = persistedRepository(ctx, provider, link.RepositoryID, link.ConnectionScope)
		if err != nil {
			unavailable = append(unavailable, map[string]any{"key": link.Key, "reason": "repository_unavailable"})
			continue
		}
		seen[pullRequestLinkStorageKey(link)] = struct{}{}
		pullRequest, getErr := provider.GetPullRequest(ctx, repository, number)
		if getErr == nil {
			pullRequests = append(pullRequests, pullRequest)
			if refreshed, linkErr := w.linkForPullRequest(ctx, workspaceID, pullRequest); linkErr == nil {
				_, _ = w.links.Link(ctx, taskID, refreshed)
			}
		}
	}
	watchIdentities := sortedWatchAssociationIdentities(watchAssociations[taskID])
	for _, storageKey := range watchIdentities {
		association := watchAssociations[taskID][storageKey]
		if _, found := seen[storageKey]; found {
			continue
		}
		repository, number, ok := parsePullRequestKey(association.Key)
		if !ok {
			continue
		}
		repository, err = persistedRepository(ctx, provider, association.RepositoryID, association.ProviderScope)
		if err != nil {
			unavailable = append(unavailable, map[string]any{"key": association.Key, "reason": "repository_unavailable"})
			continue
		}
		pullRequest, getErr := provider.GetPullRequest(ctx, repository, number)
		if getErr == nil {
			pullRequests = append(pullRequests, pullRequest)
			_ = w.watches.RefreshTaskLink(ctx, workspaceID, taskID, storageKey, pullRequest.Key(), pullRequest.URL)
		}
	}
	response := map[string]any{"pull_requests": pullRequestViews(pullRequests)}
	if len(unavailable) > 0 {
		response["unavailable_pull_requests"] = unavailable
	}
	return actionResponse(response)
}

func (w *Workflows) pullRequestAssociations(
	ctx context.Context,
	workspaceID string,
	visibleReviewKeys *[]string,
) ([]map[string]any, error) {
	if workspaceID == "" {
		return nil, forbiddenActionError("verified workspace context is required")
	}
	watchAssociations, err := w.watchOwnedAssociations(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	associations := make([]map[string]any, 0)
	visible := associationKeyFilter(visibleReviewKeys)
	page := pluginsdk.Page{Limit: 100}
	for {
		tasks, info, err := w.host.Tasks().List(ctx, pluginsdk.TaskFilter{WorkspaceIDs: []string{workspaceID}}, page)
		if err != nil {
			return nil, fmt.Errorf("list workspace tasks: %w", err)
		}
		for _, task := range tasks {
			links, err := w.links.List(ctx, task.ID)
			if err != nil {
				return nil, fmt.Errorf("list task pull request links: %w", err)
			}
			associationsByIdentity := make(map[string]watchAssociation, len(links)+len(watchAssociations[task.ID]))
			for _, link := range links {
				matchesConnection, err := w.linkMatchesConnection(ctx, workspaceID, link)
				if err != nil {
					return nil, err
				}
				if !matchesConnection || !visibleAssociationKey(visible, link.Key) {
					continue
				}
				identity := pullRequestLinkStorageKey(link)
				associationsByIdentity[identity] = watchAssociation{
					Key: link.Key, RepositoryID: link.RepositoryID,
					ProviderScope: link.ConnectionScope, Number: link.Number,
				}
			}
			for identity, association := range watchAssociations[task.ID] {
				key := association.Key
				if !visibleAssociationKey(visible, key) {
					continue
				}
				associationsByIdentity[identity] = association
			}
			for _, association := range sortedAssociations(associationsByIdentity) {
				associations = append(associations, map[string]any{
					"review_key": association.Key, "repository_id": association.RepositoryID,
					"provider_scope": association.ProviderScope, "number": association.Number,
					"task_id": task.ID, "task_title": task.Title,
				})
			}
		}
		if info == nil || !info.HasMore || info.NextCursor == "" {
			return associations, nil
		}
		page.Cursor = info.NextCursor
	}
}

func associationKeyFilter(keys *[]string) map[string]struct{} {
	if keys == nil {
		return nil
	}
	result := make(map[string]struct{}, len(*keys))
	for _, key := range *keys {
		if _, _, ok := parsePullRequestKey(key); ok {
			result[key] = struct{}{}
		}
	}
	return result
}

func visibleAssociationKey(filter map[string]struct{}, key string) bool {
	if filter == nil {
		return true
	}
	_, found := filter[key]
	return found
}

type watchAssociation struct {
	Key           string
	RepositoryID  string
	ProviderScope string
	Number        int64
}

func (w *Workflows) watchOwnedAssociations(ctx context.Context, workspaceID string) (map[string]map[string]watchAssociation, error) {
	configured, err := w.watches.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list watch-owned pull request links: %w", err)
	}
	identity, bound, err := w.connectionIdentity(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	associations := make(map[string]map[string]watchAssociation)
	for _, watch := range configured {
		if bound && identity.Binding != "" && watch.ConnectionBinding != identity.Binding {
			continue
		}
		for mapKey, link := range watch.Links {
			if !link.Owned || link.TaskID == "" || !watchLinkMatchesConnection(link, identity, bound) {
				continue
			}
			key := link.PullRequestKey
			if key == "" {
				key = mapKey
			}
			repository, number, ok := parsePullRequestKey(key)
			if !ok || link.RepositoryID == "" || link.ProviderScope == "" || link.PullRequestNumber != int64(number) ||
				(bound && !sameConnectionScope(link.ProviderScope, identity.Scope)) || repository.Namespace == "" {
				continue
			}
			if associations[link.TaskID] == nil {
				associations[link.TaskID] = make(map[string]watchAssociation)
			}
			storageKey := linkStorageKey(link)
			if storageKey == "" {
				continue
			}
			associations[link.TaskID][storageKey] = watchAssociation{
				Key: key, RepositoryID: link.RepositoryID, ProviderScope: link.ProviderScope, Number: link.PullRequestNumber,
			}
		}
	}
	return associations, nil
}

func linkStorageKey(link watches.TaskLink) string {
	key, complete := (domain.PullRequestIdentity{
		ProviderID: link.ProviderID, ProviderScope: link.ProviderScope,
		RepositoryID: link.RepositoryID, Number: link.PullRequestNumber,
	}).StorageKey()
	if !complete {
		return ""
	}
	return key
}

func hasWatchAssociationKey(associations map[string]watchAssociation, key string) bool {
	for _, association := range associations {
		if association.Key == key {
			return true
		}
	}
	return false
}

func sortedWatchAssociationIdentities(associations map[string]watchAssociation) []string {
	keys := make([]string, 0, len(associations))
	for key := range associations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func watchLinkMatchesConnection(
	link watches.TaskLink,
	identity connectionIdentity,
	bound bool,
) bool {
	if !bound {
		return true
	}
	if identity.Host == "" || identity.Scope == "" || link.ProviderID != "bitbucket" || link.ProviderHost == "" {
		return false
	}
	if !providerHostMatches(link.ProviderHost, &url.URL{Scheme: "https", Host: identity.Host}) {
		return false
	}
	if link.ConnectionScope != "" {
		return sameConnectionScope(identity.Scope, link.ConnectionScope)
	}
	if identity.Product == domain.ProductCloud {
		return true
	}
	return link.PullRequestURL != "" && urlWithinConnectionScope(link.PullRequestURL, identity.Scope)
}

func sortedAssociationKeys(keys map[string]struct{}) []string {
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortedAssociations(values map[string]watchAssociation) []watchAssociation {
	result := make([]watchAssociation, 0, len(values))
	for _, association := range values {
		result = append(result, association)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Key != result[j].Key {
			return result[i].Key < result[j].Key
		}
		return result[i].RepositoryID < result[j].RepositoryID
	})
	return result
}

func (w *Workflows) taskHasPullRequestAssociation(
	ctx context.Context,
	workspaceID, taskID string,
	lookup pullRequestLookup,
) (bool, error) {
	target, complete := (domain.PullRequestIdentity{
		ProviderID: "bitbucket", ProviderScope: lookup.ProviderScope,
		RepositoryID: lookup.RepositoryID, Number: int64(lookup.Number),
	}).StorageKey()
	if !complete {
		return false, invalidActionError("complete pull request identity is required")
	}
	links, err := w.links.List(ctx, taskID)
	if err != nil {
		return false, err
	}
	for _, link := range links {
		if pullRequestLinkStorageKey(link) != target {
			continue
		}
		matches, err := w.linkMatchesConnection(ctx, workspaceID, link)
		if err != nil {
			return false, err
		}
		if matches {
			return true, nil
		}
	}
	watchAssociations, err := w.watchOwnedAssociations(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	_, found := watchAssociations[taskID][target]
	return found, nil
}

func (w *Workflows) autoLinkTaskPullRequests(ctx context.Context, workspaceID, taskID string, provider domain.Provider) error {
	task, err := w.host.Tasks().Get(ctx, taskID)
	if err != nil || task == nil || task.ID != taskID || task.WorkspaceID != workspaceID {
		return fmt.Errorf("verified task is unavailable")
	}
	candidates, err := taskBitbucketRepositories(ctx, w.host, *task)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		remote := watches.RemoteRepository{
			ProviderID: "bitbucket", ProviderHost: candidate.repository.ProviderHost,
			ProviderScope: candidate.repository.ProviderScope, OwnerOrProject: candidate.repository.OwnerOrProject,
			ProviderRepositoryID: candidate.repository.ProviderRepositoryID,
			Name:                 providerRepositoryName(candidate.repository), CloneURL: candidate.repository.RemoteURL,
			DefaultBranch: stringValue(candidate.repository.DefaultBranch), BaseBranch: candidate.taskRepository.BaseBranch,
			HeadBranch: candidate.taskRepository.CheckoutBranch,
		}
		repository, err := domainRepository(remote)
		if err != nil {
			return fmt.Errorf("task Bitbucket repository is invalid")
		}
		pullRequests, err := provider.SearchPullRequests(ctx, domain.PullRequestQuery{Repository: repository, State: "OPEN", Limit: 100})
		if err != nil {
			return fmt.Errorf("search task pull requests: %w", err)
		}
		checkoutBranch := branchName(candidate.taskRepository.CheckoutBranch)
		for _, pullRequest := range pullRequests {
			if pullRequest.Repository.Namespace != repository.Namespace || pullRequest.Repository.Slug != repository.Slug || pullRequest.Source.Name != checkoutBranch {
				continue
			}
			link, err := w.linkForPullRequest(ctx, workspaceID, pullRequest)
			if err != nil {
				return err
			}
			if _, err := w.links.AutoLink(ctx, taskID, link); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Workflows) linkForPullRequest(ctx context.Context, workspaceID string, pullRequest domain.PullRequest) (PullRequestLink, error) {
	link := PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: pullRequest.Repository.ID,
		URL: pullRequest.URL, Number: int64(pullRequest.Number),
	}
	identity, bound, err := w.connectionIdentity(ctx, workspaceID)
	if err != nil {
		return PullRequestLink{}, err
	}
	if !bound {
		return link, nil
	}
	if identity.Product == "" || identity.Host == "" {
		return PullRequestLink{}, fmt.Errorf("Bitbucket connection is unavailable")
	}
	link.Product, link.Host, link.ConnectionScope = identity.Product, identity.Host, identity.Scope
	if _, _, err := normalizePullRequestLink(link); err != nil {
		return PullRequestLink{}, fmt.Errorf("Bitbucket pull request does not match the active connection")
	}
	return link, nil
}

func (w *Workflows) linkMatchesConnection(ctx context.Context, workspaceID string, link PullRequestLink) (bool, error) {
	identity, bound, err := w.connectionIdentity(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	if !bound {
		return true, nil
	}
	if identity.Product != link.Product || !strings.EqualFold(identity.Host, link.Host) {
		return false, nil
	}
	if link.ConnectionScope != "" {
		return sameConnectionScope(identity.Scope, link.ConnectionScope), nil
	}
	// Legacy Data Center records did not persist the context path. Match only
	// when their canonical PR URL is inside the active connection scope.
	return urlWithinConnectionScope(link.URL, identity.Scope), nil
}

func (w *Workflows) connectionIdentity(ctx context.Context, workspaceID string) (connectionIdentity, bool, error) {
	return connectionIdentityForResolver(ctx, w.resolver, workspaceID)
}
