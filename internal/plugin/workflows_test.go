package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kandev/kandev/pkg/pluginsdk"
	"github.com/stretchr/testify/require"
)

func TestDecodeActionRejectsTrailingJSONValue(t *testing.T) {
	var input listRepositoriesInput
	err := decodeAction([]byte(`{"query":"first"}{"query":"second"}`), &input)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid action body")
}

func TestWorkflows_QueueAndComposerAuthorizeLivePullRequest(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.queue", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"query":"fix","state":"OPEN"}`),
	})
	require.NoError(t, err)
	var queue struct {
		PullRequests []map[string]any `json:"pull_requests"`
	}
	require.NoError(t, json.Unmarshal(response.Body, &queue))
	require.Len(t, queue.PullRequests, 1)
	require.Equal(t, "workspace/repo#42", queue.PullRequests[0]["review_key"])
	require.Contains(t, queue.PullRequests[0]["capabilities"], "launch_task")
	require.Len(t, provider.searchQueries, 1)
	require.Equal(t, "OPEN", provider.searchQueries[0].State)

	search, err := workflows.SearchEntityReferences(context.Background(), &pluginsdk.SearchEntityReferencesRequest{Source: "bitbucket", WorkspaceID: "workspace-1", Query: "fix", Limit: 10})
	require.NoError(t, err)
	require.Len(t, search.Candidates, 1)
	candidate := search.Candidates[0]
	hostReference := map[string]any{
		"id": candidate.ProviderLocalID, "key": candidate.ProviderLocalID,
		"title": candidate.Title, "url": candidate.URL,
	}
	authorized, err := workflows.AuthorizeEntityReference(context.Background(), &pluginsdk.AuthorizeEntityReferenceRequest{
		Source: "bitbucket", WorkspaceID: "workspace-1", Purpose: "submission", Reference: hostReference,
	})
	require.NoError(t, err)
	require.True(t, authorized.Allowed)

	authorized, err = workflows.AuthorizeEntityReference(context.Background(), &pluginsdk.AuthorizeEntityReferenceRequest{
		Source: "bitbucket", WorkspaceID: "workspace-1", Purpose: "submission", Reference: candidate.Attributes,
	})
	require.NoError(t, err)
	require.True(t, authorized.Allowed)

	mismatchedCanonical := map[string]any{"id": "workspace/repo#99", "key": "workspace/repo#42"}
	authorized, err = workflows.AuthorizeEntityReference(context.Background(), &pluginsdk.AuthorizeEntityReferenceRequest{Source: "bitbucket", WorkspaceID: "workspace-1", Purpose: "submission", Reference: mismatchedCanonical})
	require.NoError(t, err)
	require.False(t, authorized.Allowed)

	tampered := map[string]any{"key": "workspace/repo#99", "repository": map[string]any{"namespace": "workspace", "slug": "repo"}, "number": 99}
	authorized, err = workflows.AuthorizeEntityReference(context.Background(), &pluginsdk.AuthorizeEntityReferenceRequest{Source: "bitbucket", WorkspaceID: "workspace-1", Purpose: "submission", Reference: tampered})
	require.NoError(t, err)
	require.False(t, authorized.Allowed)
}

func TestWorkflows_QueueUsesDeterministicCursorAcrossRepositories(t *testing.T) {
	repositoryA := domain.Repository{ID: "repo-a", ProviderScope: "https://bitbucket.org", Namespace: "workspace", Slug: "a"}
	repositoryB := domain.Repository{ID: "repo-b", ProviderScope: "https://bitbucket.org", Namespace: "workspace", Slug: "b"}
	pullRequest := func(repository domain.Repository, number int) domain.PullRequest {
		return domain.PullRequest{
			Repository: repository,
			Number:     number,
			Title:      fmt.Sprintf("Pull request %d", number),
			State:      "OPEN",
			URL:        fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", repository.Namespace, repository.Slug, number),
		}
	}
	provider := &workflowProvider{
		pullRequest:  pullRequest(repositoryA, 1),
		repositories: []domain.Repository{repositoryB, repositoryA},
		pullRequestPages: map[string]map[string]domain.PullRequestPage{
			"workspace/a": {
				"":         {PullRequests: []domain.PullRequest{pullRequest(repositoryA, 1), pullRequest(repositoryA, 2)}, NextCursor: "a-page-2"},
				"a-page-2": {PullRequests: []domain.PullRequest{pullRequest(repositoryA, 3)}},
			},
			"workspace/b": {
				"": {PullRequests: []domain.PullRequest{pullRequest(repositoryB, 1), pullRequest(repositoryB, 2)}},
			},
		},
	}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	requestPage := func(cursor string) ([]string, string) {
		body, marshalErr := json.Marshal(map[string]any{"state": "OPEN", "limit": 2, "cursor": cursor})
		require.NoError(t, marshalErr)
		response, actionErr := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
			ActionKey: "pullrequests.queue",
			Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
			Body:      body,
		})
		require.NoError(t, actionErr)
		var page struct {
			PullRequests []struct {
				ReviewKey string `json:"review_key"`
			} `json:"pull_requests"`
			NextCursor string `json:"next_cursor"`
		}
		require.NoError(t, json.Unmarshal(response.Body, &page))
		keys := make([]string, 0, len(page.PullRequests))
		for _, item := range page.PullRequests {
			keys = append(keys, item.ReviewKey)
		}
		return keys, page.NextCursor
	}

	first, cursor := requestPage("")
	require.Equal(t, []string{"workspace/a#1", "workspace/a#2"}, first)
	require.NotEmpty(t, cursor)
	second, cursor := requestPage(cursor)
	require.Equal(t, []string{"workspace/a#3", "workspace/b#1"}, second)
	require.NotEmpty(t, cursor)
	third, cursor := requestPage(cursor)
	require.Equal(t, []string{"workspace/b#2"}, third)
	require.Empty(t, cursor)
	require.Equal(t, []domain.PullRequestQuery{
		{Repository: testRepositoryWithIdentity(repositoryA), State: "OPEN", Limit: 2},
		{Repository: testRepositoryWithIdentity(repositoryA), State: "OPEN", Limit: 2, Cursor: "a-page-2"},
		{Repository: testRepositoryWithIdentity(repositoryA), State: "OPEN", Limit: 2, Cursor: "a-page-2"},
		{Repository: testRepositoryWithIdentity(repositoryB), State: "OPEN", Limit: 2},
		{Repository: testRepositoryWithIdentity(repositoryB), State: "OPEN", Limit: 2},
	}, provider.searchQueries)
}

func TestWorkflows_QueueRejectsCyclicProviderCursor(t *testing.T) {
	repository := domain.Repository{ID: "repo-a", ProviderScope: "https://bitbucket.org", Namespace: "workspace", Slug: "a"}
	provider := &workflowProvider{
		pullRequest:  domain.PullRequest{Repository: repository, Number: 1},
		repositories: []domain.Repository{repository},
		pullRequestPages: map[string]map[string]domain.PullRequestPage{
			"workspace/a": {
				"":       {NextCursor: "page-a"},
				"page-a": {NextCursor: "page-b"},
				"page-b": {NextCursor: "page-a"},
			},
		},
	}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.queue",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"state":"OPEN","limit":2}`),
	})

	require.ErrorContains(t, err, "pagination did not advance")
	require.LessOrEqual(t, len(provider.searchQueries), 3)
}

func TestWorkflows_ComposerSearchDoesNotUseCandidateLimitAsRepositoryLimit(t *testing.T) {
	first := domain.Repository{Namespace: "workspace", Slug: "empty"}
	second := domain.Repository{Namespace: "workspace", Slug: "repo"}
	pullRequest := testPullRequest()
	pullRequest.Repository = second
	provider := &workflowProvider{
		pullRequest:  pullRequest,
		repositories: []domain.Repository{first, second},
		pullRequestsByRepository: map[string][]domain.PullRequest{
			"workspace/repo": {pullRequest},
		},
	}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.SearchEntityReferences(context.Background(), &pluginsdk.SearchEntityReferencesRequest{
		Source: "bitbucket", WorkspaceID: "workspace-1", Query: "fix", Limit: 1,
	})

	require.NoError(t, err)
	require.Len(t, response.Candidates, 1)
	require.Equal(t, "workspace/repo#42", response.Candidates[0].ProviderLocalID)
	require.Equal(t, 100, provider.listRepositoryLimit)
}

func TestWorkflows_RepositoryDiscoveryReturnsOpaqueContinuation(t *testing.T) {
	first := domain.Repository{Namespace: "workspace", Slug: "one"}
	second := domain.Repository{Namespace: "workspace", Slug: "two"}
	provider := &workflowProvider{repositoryPages: map[string]domain.RepositoryPage{
		"":       {Repositories: []domain.Repository{first}, NextCursor: "page-2"},
		"page-2": {Repositories: []domain.Repository{second}},
	}}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	firstResponse, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.list",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"query":"work","limit":25}`),
	})
	require.NoError(t, err)
	require.Contains(t, string(firstResponse.Body), `"next_cursor":"page-2"`)

	secondResponse, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.list",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"query":"work","limit":25,"cursor":"page-2"}`),
	})
	require.NoError(t, err)
	require.Contains(t, string(secondResponse.Body), `"name":"two"`)
	require.Equal(t, []string{"", "page-2"}, provider.repositoryPageCursors)
}

func TestPullRequestViewIncludesCanonicalAuthor(t *testing.T) {
	pullRequest := testPullRequest()
	pullRequest.Author = "cloud-account-ada"
	view := pullRequestView(pullRequest)
	require.Equal(t, "cloud-account-ada", view["author"])
	require.Equal(t, "repo-uuid", view["repository_id"])
	require.Equal(t, "https://bitbucket.org", view["provider_scope"])
}

func TestPullRequestViewIncludesHumanDisplayMetadata(t *testing.T) {
	pullRequest := testPullRequest()
	pullRequest.Author = "cloud-account-ada"
	pullRequest.AuthorDisplayName = "Ada Lovelace"
	pullRequest.CreatedAt = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	view := pullRequestView(pullRequest)
	require.Equal(t, "cloud-account-ada", view["author"])
	require.Equal(t, "Ada Lovelace", view["author_display_name"])
	require.Equal(t, "2026-07-31T12:00:00Z", view["created_at"])
}

func TestWorkflows_PullRequestAssociationsPaginatesWorkspaceTasksAndSkipsEmptyLinks(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {
			tasks: []pluginsdk.Task{{ID: "task-1", Title: "Fix auth"}, {ID: "task-without-link", Title: "Unrelated"}},
			info:  &pluginsdk.PageInfo{HasMore: true, NextCursor: "page-2"},
		},
		"page-2": {tasks: []pluginsdk.Task{{ID: "task-2", Title: "Review race"}}},
	}}
	host := &associationHost{connectionHost: newConnectionHost(), tasks: tasks}
	provider := &workflowProvider{}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: "workspace/repo#42", RepositoryID: "workspace/repo", URL: "https://bitbucket.org/workspace/repo/pull-requests/42", Number: 42,
	})
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-2", PullRequestLink{
		Key: "workspace/repo#43", RepositoryID: "workspace/repo", URL: "https://bitbucket.org/workspace/repo/pull-requests/43", Number: 43,
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[
		{"review_key":"workspace/repo#42","repository_id":"workspace/repo","provider_scope":"https://bitbucket.org","number":42,"task_id":"task-1","task_title":"Fix auth"},
		{"review_key":"workspace/repo#43","repository_id":"workspace/repo","provider_scope":"https://bitbucket.org","number":43,"task_id":"task-2","task_title":"Review race"}
	]}`, string(response.Body))
	require.Equal(t, []string{"", "page-2"}, tasks.cursors)
	require.Equal(t, []pluginsdk.TaskFilter{
		{WorkspaceIDs: []string{"workspace-1"}}, {WorkspaceIDs: []string{"workspace-1"}},
	}, tasks.filters)
	require.Empty(t, provider.searchQueries, "association lookup must not call Bitbucket")
	require.Zero(t, provider.getPullRequestCalls, "association lookup must not call Bitbucket")
}

func TestWorkflows_PullRequestAssociationsHonorsVisibleReviewKeys(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "task-1", Title: "Fix auth"}, {ID: "task-2", Title: "Other"}}},
	}}
	workflows, err := NewWorkflows(
		&associationHost{connectionHost: newConnectionHost(), tasks: tasks},
		staticResolver{provider: &workflowProvider{}},
	)
	require.NoError(t, err)
	for taskID, number := range map[string]int64{"task-1": 42, "task-2": 43} {
		_, err = workflows.links.Link(context.Background(), taskID, PullRequestLink{
			Key: fmt.Sprintf("workspace/repo#%d", number), RepositoryID: "workspace/repo",
			URL: fmt.Sprintf("https://bitbucket.org/workspace/repo/pull-requests/%d", number), Number: number,
		})
		require.NoError(t, err)
	}

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"review_keys":["workspace/repo#43"]}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[{"review_key":"workspace/repo#43","repository_id":"workspace/repo","provider_scope":"https://bitbucket.org","number":43,"task_id":"task-2","task_title":"Other"}]}`, string(response.Body))
}

func TestWorkflows_PullRequestAssociationsHidesLinksFromPreviousConnection(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "task-1", Title: "Old host"}}},
	}}
	resolver := &connectionSettingsResolver{
		staticResolver: staticResolver{provider: &workflowProvider{}},
		settings:       ConnectionSettings{Product: domain.ProductDataCenter, BaseURL: "https://new.example.test"},
		found:          true,
	}
	workflows, err := NewWorkflows(
		&associationHost{connectionHost: newConnectionHost(), tasks: tasks},
		resolver,
	)
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: "PROJECT/repo#42", RepositoryID: "PROJECT/repo",
		URL: "https://old.example.test/projects/PROJECT/repos/repo/pull-requests/42", Number: 42,
		Product: domain.ProductDataCenter, Host: "old.example.test",
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[]}`, string(response.Body))
}

func TestWorkflows_PullRequestAssociationsHidesLinksFromPreviousDataCenterContext(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "task-1", Title: "Old context"}}},
	}}
	resolver := &connectionSettingsResolver{
		staticResolver: staticResolver{provider: &workflowProvider{}},
		settings: ConnectionSettings{
			Product: domain.ProductDataCenter,
			BaseURL: "https://bitbucket.example.test/current",
		},
		found: true,
	}
	workflows, err := NewWorkflows(
		&associationHost{connectionHost: newConnectionHost(), tasks: tasks},
		resolver,
	)
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: "PROJECT/repo#42", RepositoryID: "PROJECT/repo",
		URL: "https://bitbucket.example.test/previous/projects/PROJECT/repos/repo/pull-requests/42", Number: 42,
		Product: domain.ProductDataCenter, Host: "bitbucket.example.test",
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[]}`, string(response.Body))
}

func TestWorkflows_PullRequestAssociationsHidesWatchLinksFromPreviousConnection(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "watch-task", Title: "Old watch"}}},
	}}
	resolver := &connectionSettingsResolver{
		staticResolver: staticResolver{provider: &workflowProvider{}},
		settings:       ConnectionSettings{Product: domain.ProductDataCenter, BaseURL: "https://new.example.test"},
		found:          true,
	}
	workflows, err := NewWorkflows(
		&associationHost{connectionHost: newConnectionHost(), tasks: tasks},
		resolver,
	)
	require.NoError(t, err)
	_, err = workflows.watches.Create(context.Background(), watches.Watch{
		ID: "watch-1", WorkspaceID: "workspace-1",
		Links: map[string]watches.TaskLink{
			"PROJECT/repo#42": {
				PullRequestKey: "PROJECT/repo#42", TaskID: "watch-task", Owned: true,
				ProviderID: "bitbucket", ProviderHost: "old.example.test", ProviderScope: "https://old.example.test",
				RepositoryID: "repo-42", PullRequestNumber: 42,
			},
		},
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[]}`, string(response.Body))
}

func TestWorkflows_PullRequestAssociationsAcceptsCurrentOriginWatchLink(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "watch-task", Title: "Current watch"}}},
	}}
	resolver := &connectionSettingsResolver{
		staticResolver: staticResolver{provider: &workflowProvider{}},
		settings:       ConnectionSettings{Product: domain.ProductCloud},
		found:          true,
	}
	workflows, err := NewWorkflows(
		&associationHost{connectionHost: newConnectionHost(), tasks: tasks},
		resolver,
	)
	require.NoError(t, err)
	_, err = workflows.watches.Create(context.Background(), watches.Watch{
		ID: "watch-1", WorkspaceID: "workspace-1",
		Links: map[string]watches.TaskLink{
			"workspace/repo#42": {
				PullRequestKey: "workspace/repo#42", TaskID: "watch-task", Owned: true,
				ProviderID: "bitbucket", ProviderHost: "https://bitbucket.org", ProviderScope: "https://bitbucket.org",
				RepositoryID: "repo-uuid", PullRequestNumber: 42,
			},
		},
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[{"review_key":"workspace/repo#42","repository_id":"repo-uuid","provider_scope":"https://bitbucket.org","number":42,"task_id":"watch-task","task_title":"Current watch"}]}`, string(response.Body))
}

func TestWorkflows_PullRequestAssociationsScopesWatchLinksToDataCenterContext(t *testing.T) {
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "old-task", Title: "Old context"}, {ID: "current-task", Title: "Current context"}}},
	}}
	resolver := &connectionSettingsResolver{
		staticResolver: staticResolver{provider: &workflowProvider{}},
		settings: ConnectionSettings{
			Product: domain.ProductDataCenter,
			BaseURL: "https://bitbucket.example.test/current",
		},
		found: true,
	}
	workflows, err := NewWorkflows(
		&associationHost{connectionHost: newConnectionHost(), tasks: tasks},
		resolver,
	)
	require.NoError(t, err)
	_, err = workflows.watches.Create(context.Background(), watches.Watch{
		ID: "watch-1", WorkspaceID: "workspace-1",
		Links: map[string]watches.TaskLink{
			"PROJECT/old#41": {
				PullRequestKey: "PROJECT/old#41", TaskID: "old-task", Owned: true,
				ProviderID: "bitbucket", ProviderHost: "https://bitbucket.example.test", ProviderScope: "https://bitbucket.example.test/previous",
				RepositoryID: "repo-old", PullRequestNumber: 41,
				ConnectionScope: "https://bitbucket.example.test/previous",
			},
			"PROJECT/current#42": {
				PullRequestKey: "PROJECT/current#42", TaskID: "current-task", Owned: true,
				ProviderID: "bitbucket", ProviderHost: "https://bitbucket.example.test", ProviderScope: "https://bitbucket.example.test/current",
				RepositoryID: "repo-current", PullRequestNumber: 42,
				ConnectionScope: "https://bitbucket.example.test/current",
			},
		},
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[{"review_key":"PROJECT/current#42","repository_id":"repo-current","provider_scope":"https://bitbucket.example.test/current","number":42,"task_id":"current-task","task_title":"Current context"}]}`, string(response.Body))
}

func TestWorkflows_WatchOwnedLinksAppearInTaskAndWorkspaceAssociations(t *testing.T) {
	ctx := context.Background()
	tasks := &associationTaskReader{pages: map[string]associationTaskPage{
		"": {tasks: []pluginsdk.Task{{ID: "watch-task", Title: "Watch-created task"}}},
	}}
	host := &associationHost{connectionHost: newConnectionHost(), tasks: tasks}
	pullRequest := testPullRequest()
	provider := &workflowProvider{pullRequest: pullRequest}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.watches.Create(ctx, watches.Watch{
		ID: "watch-1", WorkspaceID: "workspace-1",
		Links: map[string]watches.TaskLink{
			pullRequest.Key(): {
				PullRequestKey: pullRequest.Key(), TaskID: "watch-task", Owned: true,
				ProviderID: "bitbucket", ProviderScope: pullRequest.Repository.ProviderScope,
				RepositoryID: pullRequest.Repository.ID, PullRequestNumber: int64(pullRequest.Number),
			},
		},
	})
	require.NoError(t, err)
	_, err = workflows.links.Link(ctx, "watch-task", PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: pullRequest.Repository.ID, URL: pullRequest.URL, Number: int64(pullRequest.Number),
	})
	require.NoError(t, err)
	workflows, err = NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err, "persisted watch links must survive a plugin restart")

	taskResponse, err := workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "watch-task"},
	})
	require.NoError(t, err)
	var taskResult struct {
		PullRequests []map[string]any `json:"pull_requests"`
	}
	require.NoError(t, json.Unmarshal(taskResponse.Body, &taskResult))
	require.Len(t, taskResult.PullRequests, 1, "manual and watch-owned associations must be deduplicated")
	require.Equal(t, pullRequest.Key(), taskResult.PullRequests[0]["review_key"])

	workspaceResponse, err := workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[{"review_key":"workspace/repo#42","repository_id":"repo-uuid","provider_scope":"https://bitbucket.org","number":42,"task_id":"watch-task","task_title":"Watch-created task"}]}`, string(workspaceResponse.Body))

	_, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.unlink",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "watch-task"},
		Body:      []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42}`),
	})
	require.NoError(t, err)

	taskResponse, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "watch-task"},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"pull_requests":[]}`, string(taskResponse.Body))
	workspaceResponse, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.associations",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"associations":[]}`, string(workspaceResponse.Body))
}

func TestWorkflows_TaskScopedExplicitGetRequiresManualOrWatchAssociation(t *testing.T) {
	ctx := context.Background()
	pullRequest := testPullRequest()
	provider := &workflowProvider{pullRequest: pullRequest}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	request := func(taskID string) *pluginsdk.PluginActionRequest {
		return &pluginsdk.PluginActionRequest{
			ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: taskID},
			Body: []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42}`),
		}
	}

	_, err = workflows.HandleAction(ctx, request("unassociated-task"))
	require.Error(t, err)
	require.Zero(t, provider.getPullRequestCalls, "authorization must precede the live pull request fetch")

	_, err = workflows.links.Link(ctx, "manual-task", PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: pullRequest.Repository.ID, URL: pullRequest.URL, Number: int64(pullRequest.Number),
		Product: domain.ProductCloud, Host: "bitbucket.org", ConnectionScope: pullRequest.Repository.ProviderScope,
	})
	require.NoError(t, err)
	_, err = workflows.HandleAction(ctx, request("manual-task"))
	require.NoError(t, err)

	_, err = workflows.watches.Create(ctx, watches.Watch{
		ID: "watch-1", WorkspaceID: "workspace-1",
		Links: map[string]watches.TaskLink{
			pullRequest.Key(): {
				PullRequestKey: pullRequest.Key(), TaskID: "watch-task", Owned: true,
				ProviderID: "bitbucket", ProviderScope: pullRequest.Repository.ProviderScope,
				RepositoryID: pullRequest.Repository.ID, PullRequestNumber: int64(pullRequest.Number),
			},
		},
	})
	require.NoError(t, err)
	_, err = workflows.HandleAction(ctx, request("watch-task"))
	require.NoError(t, err)

	_, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "unassociated-task"},
		Body: []byte(`{"review_key":"workspace/repo#42"}`),
	})
	require.NoError(t, err, "workspace-scoped inspection remains available")
}

func TestWorkflows_PullRequestGetDoesNotMasqueradeReviewFailureAsEmptyData(t *testing.T) {
	provider := &workflowProvider{
		pullRequest: testPullRequest(),
		reviewErr:   errors.New("diff endpoint unavailable"),
	}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.inspect",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42}`),
	})

	require.ErrorContains(t, err, "get pull request review")
	require.Nil(t, response)
}

func TestWorkflows_PullRequestGetHonorsReviewProjection(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42","include":["participants","thread_count","status","viewer"]}`),
	})

	require.NoError(t, err)
	require.Equal(t, []domain.ReviewProjection{{Participants: true, Statuses: true, Viewer: true, ThreadCount: true}}, provider.reviewProjections)
}

func TestWorkflows_TaskGetAutoLinksOpenPullRequestForVerifiedCheckoutBranch(t *testing.T) {
	host := autoLinkHost(
		[]pluginsdk.TaskRepository{{RepositoryID: "repo-1", CheckoutBranch: "refs/heads/feature/auth"}},
		[]pluginsdk.Repository{bitbucketHostRepository("repo-1", "workspace", "repo")},
	)
	pullRequest := testPullRequest()
	pullRequest.Source.Name = "feature/auth"
	provider := &workflowProvider{pullRequest: pullRequest}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})

	require.NoError(t, err)
	require.Contains(t, string(response.Body), pullRequest.Key())
	links, err := workflows.links.List(context.Background(), "task-1")
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, pullRequest.Key(), links[0].Key)
	require.Len(t, provider.searchQueries, 1)
	require.Equal(t, pullRequest.Repository.Namespace, provider.searchQueries[0].Repository.Namespace)
	require.Equal(t, pullRequest.Repository.Slug, provider.searchQueries[0].Repository.Slug)
	require.Equal(t, "OPEN", provider.searchQueries[0].State)
	require.Equal(t, 100, provider.searchQueries[0].Limit)
}

func TestWorkflows_TaskGetKeepsAssociationAcrossRepositoryRename(t *testing.T) {
	ctx := context.Background()
	host := &associationHost{
		connectionHost: newConnectionHost(),
		tasks: &associationTaskReader{pages: map[string]associationTaskPage{
			"": {tasks: []pluginsdk.Task{{ID: "task-1", Title: "Renamed repository"}}},
		}},
	}
	pullRequest := testPullRequest()
	pullRequest.Repository.Namespace = "new-workspace"
	pullRequest.Repository.Slug = "new-repo"
	pullRequest.URL = "https://bitbucket.org/new-workspace/new-repo/pull-requests/42"
	provider := &workflowProvider{pullRequest: pullRequest, repositories: []domain.Repository{pullRequest.Repository}}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.links.Link(ctx, "task-1", PullRequestLink{
		Key: "old-workspace/old-repo#42", RepositoryID: pullRequest.Repository.ID,
		URL: "https://bitbucket.org/old-workspace/old-repo/pull-requests/42", Number: 42,
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})
	require.NoError(t, err)
	require.Contains(t, string(response.Body), `"review_key":"new-workspace/new-repo#42"`)
	require.Equal(t, []string{""}, provider.repositoryQueries, "immutable lookup must not filter by a stale repository slug")
	links, err := workflows.links.List(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, "new-workspace/new-repo#42", links[0].Key)
	require.Equal(t, pullRequest.URL, links[0].URL)
}

func TestWorkflows_TaskGetDoesNotAutoRelinkAfterExplicitUnlink(t *testing.T) {
	host := autoLinkHost(
		[]pluginsdk.TaskRepository{{RepositoryID: "repo-1", CheckoutBranch: "feature/auth"}},
		[]pluginsdk.Repository{bitbucketHostRepository("repo-1", "workspace", "repo")},
	)
	pullRequest := testPullRequest()
	pullRequest.Source.Name = "feature/auth"
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: pullRequest}})
	require.NoError(t, err)
	ctx := context.Background()

	_, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})
	require.NoError(t, err)
	_, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.unlink",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
		Body:      []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42}`),
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"pull_requests":[]}`, string(response.Body))
}

func TestWorkflows_TaskGetDoesNotAutoLinkDifferentCheckoutBranch(t *testing.T) {
	host := autoLinkHost(
		[]pluginsdk.TaskRepository{{RepositoryID: "repo-1", CheckoutBranch: "feature/wanted"}},
		[]pluginsdk.Repository{bitbucketHostRepository("repo-1", "workspace", "repo")},
	)
	pullRequest := testPullRequest()
	pullRequest.Source.Name = "feature/other"
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: pullRequest}})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"pull_requests":[]}`, string(response.Body))
}

func TestWorkflows_TaskGetAutoLinksAcrossVerifiedTaskRepositories(t *testing.T) {
	host := autoLinkHost(
		[]pluginsdk.TaskRepository{
			{RepositoryID: "repo-1", CheckoutBranch: "feature/one"},
			{RepositoryID: "repo-2", CheckoutBranch: "refs/heads/feature/two"},
		},
		[]pluginsdk.Repository{
			bitbucketHostRepository("repo-1", "workspace", "one"),
			bitbucketHostRepository("repo-2", "workspace", "two"),
		},
	)
	first := testPullRequest()
	first.Repository.Slug = "one"
	first.Number = 41
	first.Source.Name = "feature/one"
	first.URL = "https://bitbucket.org/workspace/one/pull-requests/41"
	second := testPullRequest()
	second.Repository.ID = "repo-uuid-two"
	second.Repository.Slug = "two"
	second.Number = 42
	second.Source.Name = "feature/two"
	second.URL = "https://bitbucket.org/workspace/two/pull-requests/42"
	provider := &workflowProvider{
		pullRequest:  first,
		repositories: []domain.Repository{first.Repository, second.Repository},
		pullRequestsByRepository: map[string][]domain.PullRequest{
			"workspace/one": {first},
			"workspace/two": {second},
		},
	}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})

	require.NoError(t, err)
	var result struct {
		PullRequests []map[string]any `json:"pull_requests"`
	}
	require.NoError(t, json.Unmarshal(response.Body, &result))
	require.Len(t, result.PullRequests, 2)
	links, err := workflows.links.List(context.Background(), "task-1")
	require.NoError(t, err)
	require.Len(t, links, 2)
}

func TestWorkflows_TaskGetPreservesExistingLinkWhenBranchDetectionFails(t *testing.T) {
	host := autoLinkHost(
		[]pluginsdk.TaskRepository{{RepositoryID: "repo-1", CheckoutBranch: "feature/auth"}},
		[]pluginsdk.Repository{bitbucketHostRepository("repo-1", "workspace", "repo")},
	)
	pullRequest := testPullRequest()
	provider := &workflowProvider{pullRequest: pullRequest, searchErr: errors.New("provider unavailable")}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: "workspace/repo", URL: pullRequest.URL, Number: int64(pullRequest.Number),
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
	})

	require.NoError(t, err)
	require.Contains(t, string(response.Body), pullRequest.Key())
}

func autoLinkHost(taskRepositories []pluginsdk.TaskRepository, repositories []pluginsdk.Repository) *scopedConnectionHost {
	return &scopedConnectionHost{
		connectionHost: newConnectionHost(),
		tasks: &taskReader{task: &pluginsdk.Task{
			ID: "task-1", WorkspaceID: "workspace-1", Title: "Task", Repositories: taskRepositories,
		}},
		repositories: &repositoryReader{repositories: repositories},
	}
}

func bitbucketHostRepository(id, namespace, slug string) pluginsdk.Repository {
	defaultBranch := "main"
	return pluginsdk.Repository{
		ID: id, WorkspaceID: "workspace-1", Name: namespace + "/" + slug, SourceType: "provider", ProviderID: "bitbucket",
		ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: namespace, ProviderRepositoryID: "uuid-" + id,
		ProviderName: slug, RemoteURL: "https://bitbucket.org/" + namespace + "/" + slug + ".git", DefaultBranch: &defaultBranch,
	}
}

type associationTaskPage struct {
	tasks []pluginsdk.Task
	info  *pluginsdk.PageInfo
}

type associationTaskReader struct {
	pages   map[string]associationTaskPage
	cursors []string
	filters []pluginsdk.TaskFilter
}

func (r *associationTaskReader) List(_ context.Context, filter pluginsdk.TaskFilter, page pluginsdk.Page) ([]pluginsdk.Task, *pluginsdk.PageInfo, error) {
	r.filters = append(r.filters, filter)
	r.cursors = append(r.cursors, page.Cursor)
	result := r.pages[page.Cursor]
	return result.tasks, result.info, nil
}

func (*associationTaskReader) Get(context.Context, string) (*pluginsdk.Task, error) {
	return nil, nil
}

func (*associationTaskReader) Create(context.Context, pluginsdk.CreateTaskInput) (*pluginsdk.Task, error) {
	return nil, nil
}

func (*associationTaskReader) Update(context.Context, pluginsdk.UpdateTaskInput) (*pluginsdk.Task, error) {
	return nil, nil
}

func (*associationTaskReader) Move(context.Context, pluginsdk.MoveTaskInput) (*pluginsdk.MoveTaskOutcome, error) {
	return nil, errors.New("associationTaskReader.Move is not implemented")
}

type associationHost struct {
	*connectionHost
	tasks *associationTaskReader
}

func (h *associationHost) Tasks() pluginsdk.TaskReader { return h.tasks }

func TestWorkflows_LinkAndUnlinkDoNotDeleteTask(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	host := newConnectionHost()
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	link, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.link", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "manual-task"}, Body: []byte(`{"review_key":"workspace/repo#42"}`),
	})
	require.NoError(t, err)
	require.Contains(t, string(link.Body), "workspace/repo#42")
	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.unlink", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "manual-task"}, Body: []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42}`),
	})
	require.NoError(t, err)
	require.Empty(t, host.secrets, "links must not perform task or secret cleanup")

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "manual-task"}, Body: []byte(`{"view":"task"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"pull_requests":[]}`, string(response.Body))
}

func TestWorkflows_TaskLinkDoesNotRetargetAfterDataCenterHostChanges(t *testing.T) {
	pullRequest := testPullRequest()
	pullRequest.Repository = domain.Repository{Namespace: "ENG", Slug: "repo"}
	pullRequest.Number = 42
	pullRequest.URL = "https://bitbucket-one.example.test/projects/ENG/repos/repo/pull-requests/42"
	provider := &workflowProvider{pullRequest: pullRequest}
	resolver := &connectionSettingsResolver{
		staticResolver: staticResolver{provider: provider},
		settings:       ConnectionSettings{Product: domain.ProductDataCenter, BaseURL: "https://bitbucket-one.example.test"},
		found:          true,
	}
	workflows, err := NewWorkflows(newConnectionHost(), resolver)
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.link", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{"review_key":"ENG/repo#42"}`),
	})
	require.NoError(t, err)
	provider.getPullRequestCalls = 0
	resolver.settings.BaseURL = "https://bitbucket-two.example.test"

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{"view":"task"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"pull_requests":[],"unavailable_pull_requests":[{"key":"ENG/repo#42","reason":"connection_changed"}]}`, string(response.Body))
	require.Zero(t, provider.getPullRequestCalls, "a link must not be fetched from a new Bitbucket host")
}

func TestWorkflows_TaskLinkDoesNotRetargetAfterRepositoryRecreation(t *testing.T) {
	pullRequest := testPullRequest()
	provider := &workflowProvider{pullRequest: pullRequest, repositories: []domain.Repository{pullRequest.Repository}}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: "deleted-repository-uuid",
		URL: pullRequest.URL, Number: int64(pullRequest.Number),
		Product: domain.ProductCloud, Host: "bitbucket.org", ConnectionScope: "https://bitbucket.org",
	})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{"view":"task"}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"pull_requests":[],"unavailable_pull_requests":[{"key":"workspace/repo#42","reason":"repository_unavailable"}]}`, string(response.Body))
	require.Zero(t, provider.getPullRequestCalls, "a recreated repository at the same path must not receive the old link")
}

func TestWorkflows_HidesProviderSecretsFromCredentialRPC(t *testing.T) {
	provider := &workflowProvider{credentialErr: errors.New("provider rejected top-secret-token")}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.ResolveGitCredential(context.Background(), &pluginsdk.ResolveGitCredentialRequest{ProviderID: "bitbucket", WorkspaceID: "workspace-1", TaskID: "task", SessionID: "session", RepositoryID: "repo", Host: "bitbucket.org", Path: "/workspace/repo.git"})
	require.ErrorIs(t, err, ErrCredentialUnavailable)
	require.NotContains(t, err.Error(), "top-secret-token")
}

func TestWorkflows_UsesConnectionBindingWithoutResolvingProviderCredential(t *testing.T) {
	provider := &workflowProvider{credentialErr: errors.New("credential must not be resolved")}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider, binding: "connection:9"})
	require.NoError(t, err)

	response, err := workflows.GetGitCredentialBinding(context.Background(), &pluginsdk.GitCredentialBindingRequest{
		ProviderID: "bitbucket", WorkspaceID: "workspace-1", TaskID: "task", SessionID: "session", RepositoryID: "repo", Host: "bitbucket.org", Path: "/workspace/repo.git",
	})

	require.NoError(t, err)
	require.Equal(t, "connection:9", response.Binding)
}

func TestWorkflows_RejectsUnsupportedReviewMutationBeforeProviderCall(t *testing.T) {
	pullRequest := testPullRequest()
	provider := &workflowProvider{pullRequest: pullRequest, capabilities: domain.Capabilities{domain.CapabilityPullRequests: true}}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: pullRequest.Repository.ID,
		URL: pullRequest.URL, Number: int64(pullRequest.Number), Product: domain.ProductCloud,
		Host: "bitbucket.org", ConnectionScope: pullRequest.Repository.ProviderScope,
	})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "reviews.action", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42,"operation":"merge"}`),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support merge")
	require.Zero(t, provider.actions)
}

func TestWorkflows_ReviewMutationCannotRetargetRecreatedRepository(t *testing.T) {
	pullRequest := testPullRequest()
	pullRequest.Repository.ID = "repo-uuid-new"
	provider := &workflowProvider{pullRequest: pullRequest}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	_, err = workflows.links.Link(context.Background(), "task-1", PullRequestLink{
		Key: pullRequest.Key(), RepositoryID: "repo-uuid-old",
		URL: pullRequest.URL, Number: int64(pullRequest.Number), Product: domain.ProductCloud,
		Host: "bitbucket.org", ConnectionScope: pullRequest.Repository.ProviderScope,
	})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "reviews.action",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
		Body:      []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid-new","number":42,"operation":"merge"}`),
	})

	require.ErrorContains(t, err, "not associated with the verified task")
	require.Zero(t, provider.getPullRequestCalls)
	require.Zero(t, provider.actions)
}

func TestWorkflows_RejectsCredentialBearingRepositoryInspectionURL(t *testing.T) {
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: &workflowProvider{pullRequest: testPullRequest()}})
	require.NoError(t, err)
	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"url":"https://secret-token@bitbucket.org/workspace/repo.git"}`),
	})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-token")
}

func TestWorkflows_UsesProviderURLInspectors(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"url":"https://bitbucket.org/workspace/repo"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.org/workspace/repo", provider.inspectedRepositoryURL)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"review_key":"https://bitbucket.org/workspace/repo/pull-requests/42"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.org/workspace/repo/pull-requests/42", provider.inspectedPullRequestURL)
	require.Contains(t, string(response.Body), "workspace/repo#42")
}

func TestWorkflows_RepositoryInspectionReturnsNoMatchForAnotherProvider(t *testing.T) {
	provider := &workflowProvider{
		pullRequest:           testPullRequest(),
		repositoryInspectErr:  errors.New("not a Bitbucket repository URL"),
		pullRequestInspectErr: errors.New("not a Bitbucket pull request URL"),
	}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.inspect",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"url":"https://git.example.test/team/repository"}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"matched":false}`, string(response.Body))
}

func TestWorkflows_UnconfiguredRepositoryInspectionReturnsNoMatch(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.inspect",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"url":"https://github.com/kdlbs/kandev"}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"matched":false}`, string(response.Body))
}

func TestWorkflows_RepositoryInspectionFetchesPullRequestURLMetadata(t *testing.T) {
	pullRequest := testPullRequest()
	pullRequest.Source = domain.Branch{Name: "feature/fix-auth"}
	pullRequest.Destination = domain.Branch{Name: "main"}
	provider := &workflowProvider{pullRequest: pullRequest, repositoryInspectErr: errors.New("not a repository URL")}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"url":"https://bitbucket.org/workspace/repo/pull-requests/42"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.org/workspace/repo/pull-requests/42", provider.inspectedRepositoryURL)
	require.Equal(t, "https://bitbucket.org/workspace/repo/pull-requests/42", provider.inspectedPullRequestURL)
	require.Equal(t, 1, provider.getPullRequestCalls)

	var result struct {
		Repository  map[string]any `json:"repository"`
		PullRequest map[string]any `json:"pull_request"`
		BaseBranch  string         `json:"base_branch"`
		HeadBranch  string         `json:"head_branch"`
	}
	require.NoError(t, json.Unmarshal(response.Body, &result))
	require.Equal(t, "workspace/repo", result.Repository["id"])
	require.Equal(t, "main", result.BaseBranch)
	require.Equal(t, "feature/fix-auth", result.HeadBranch)
	require.Equal(t, "main", result.Repository["base_branch"])
	require.Equal(t, "feature/fix-auth", result.Repository["head_branch"])
	require.Equal(t, "workspace/repo#42", result.PullRequest["review_key"])
	require.Equal(t, float64(42), result.PullRequest["number"])
	require.Equal(t, "Fix auth", result.PullRequest["title"])
	require.Equal(t, "https://bitbucket.org/workspace/repo/pull-requests/42", result.PullRequest["url"])
}

func TestWorkflows_CreatePullRequestDerivesRepositoryAndSourceFromVerifiedTask(t *testing.T) {
	host := newTaskHost()
	host.tasks.task = &pluginsdk.Task{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Host task title", Description: "Host task description",
		Repositories: []pluginsdk.TaskRepository{{RepositoryID: "repository-1", BaseBranch: "main", CheckoutBranch: "refs/heads/fix-auth"}},
	}
	defaultBranch := "main"
	host.repositories.repositories = []pluginsdk.Repository{{
		ID: "repository-1", WorkspaceID: "workspace-1", Name: "repo", SourceType: "provider", ProviderID: "bitbucket",
		ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-uuid",
		RemoteURL: "https://bitbucket.org/workspace/repo.git", DefaultBranch: &defaultBranch,
	}}
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
		Body: []byte(`{"title":"Override title","destination":"release"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "workspace", provider.createdPullRequest.Repository.Namespace)
	require.Equal(t, "repo", provider.createdPullRequest.Repository.Slug)
	require.Equal(t, "https://bitbucket.org/workspace/repo.git", provider.createdPullRequest.Repository.CloneURL.String())
	require.Equal(t, "Override title", provider.createdPullRequest.Title)
	require.Equal(t, "Host task description", provider.createdPullRequest.Description)
	require.Equal(t, "fix-auth", provider.createdPullRequest.Source)
	require.Equal(t, "release", provider.createdPullRequest.Destination)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"},
		Body: []byte(`{"source":"attacker-branch"}`),
	})
	require.Error(t, err, "browser input cannot select a source checkout")
}

func TestWorkflows_CreatePullRequestUsesVerifiedSessionHeadBranch(t *testing.T) {
	host := newTaskHost()
	host.tasks.task = &pluginsdk.Task{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Host task title",
		Repositories: []pluginsdk.TaskRepository{{RepositoryID: "repository-1", BaseBranch: "main"}},
	}
	defaultBranch := "main"
	host.repositories.repositories = []pluginsdk.Repository{{
		ID: "repository-1", WorkspaceID: "workspace-1", Name: "repo", SourceType: "provider", ProviderID: "bitbucket",
		ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-uuid",
		RemoteURL: "https://bitbucket.org/workspace/repo.git", DefaultBranch: &defaultBranch,
	}}
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create",
		Context: pluginsdk.VerifiedActionContext{
			WorkspaceID: "workspace-1", TaskID: "task-1", SessionID: "session-1",
			RepositoryID: "repository-1", HeadBranch: "refs/heads/feature/native-create",
		},
		Body: []byte(`{}`),
	})

	require.NoError(t, err)
	require.Equal(t, "feature/native-create", provider.createdPullRequest.Source)
	require.Equal(t, "main", provider.createdPullRequest.Destination)
}

func TestWorkflows_CreatePullRequestSelectsVerifiedRepositoryIDForMultiRepoTask(t *testing.T) {
	host := newTaskHost()
	host.tasks.task = &pluginsdk.Task{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Host task",
		Repositories: []pluginsdk.TaskRepository{
			{RepositoryID: "repository-1", BaseBranch: "main", CheckoutBranch: "feature-one"},
			{RepositoryID: "repository-2", BaseBranch: "develop", CheckoutBranch: "feature-two"},
		},
	}
	defaultBranch := "main"
	host.repositories.repositories = []pluginsdk.Repository{
		{ID: "repository-1", WorkspaceID: "workspace-1", Name: "one", SourceType: "provider", ProviderID: "bitbucket", ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-one", RemoteURL: "https://bitbucket.org/workspace/one.git", DefaultBranch: &defaultBranch},
		{ID: "repository-2", WorkspaceID: "workspace-1", Name: "two", SourceType: "provider", ProviderID: "bitbucket", ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-two", RemoteURL: "https://bitbucket.org/workspace/two.git", DefaultBranch: &defaultBranch},
	}
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1", RepositoryID: "repository-2"},
		Body:      []byte(`{}`),
	})

	require.NoError(t, err)
	require.Equal(t, "two", provider.createdPullRequest.Repository.Slug)
	require.Equal(t, "feature-two", provider.createdPullRequest.Source)
	require.Equal(t, "develop", provider.createdPullRequest.Destination)
}

func TestWorkflows_CreatePullRequestRejectsAmbiguousOrUnattachedVerifiedRepository(t *testing.T) {
	host := newTaskHost()
	host.tasks.task = &pluginsdk.Task{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Host task",
		Repositories: []pluginsdk.TaskRepository{
			{RepositoryID: "repository-1", BaseBranch: "main", CheckoutBranch: "feature-one"},
			{RepositoryID: "repository-2", BaseBranch: "main", CheckoutBranch: "feature-two"},
		},
	}
	host.repositories.repositories = []pluginsdk.Repository{
		{ID: "repository-1", WorkspaceID: "workspace-1", Name: "one", SourceType: "provider", ProviderID: "bitbucket", ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-one", RemoteURL: "https://bitbucket.org/workspace/one.git"},
		{ID: "repository-2", WorkspaceID: "workspace-1", Name: "two", SourceType: "provider", ProviderID: "bitbucket", ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-two", RemoteURL: "https://bitbucket.org/workspace/two.git"},
	}
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: testPullRequest()}})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{}`),
	})
	require.ErrorContains(t, err, "exactly one")

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1", RepositoryID: "repository-attacker"}, Body: []byte(`{}`),
	})
	require.ErrorContains(t, err, "verified repository")
}

func TestWorkflows_CreatePullRequestPersistsTaskAssociation(t *testing.T) {
	host := createPullRequestHost()
	pullRequest := testPullRequest()
	provider := &workflowProvider{pullRequest: pullRequest}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{}`),
	})
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(response.Body, &result))
	require.Equal(t, true, result["linked"])
	require.NotContains(t, result, "association_error")
	links, err := workflows.links.List(context.Background(), "task-1")
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, pullRequest.Key(), links[0].Key)
}

func TestWorkflows_CreatePullRequestReportsAssociationFailureWithoutRetryableError(t *testing.T) {
	host := createPullRequestHost()
	host.setStateErr = errors.New("state unavailable with sensitive implementation detail")
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{}`),
	})

	require.NoError(t, err, "external creation succeeded, so an association failure must not invite a duplicate retry")
	var result map[string]any
	require.NoError(t, json.Unmarshal(response.Body, &result))
	require.Equal(t, false, result["linked"])
	require.Equal(t, "task association could not be saved", result["association_error"])
	require.NotContains(t, string(response.Body), "sensitive implementation detail")
	require.NotEmpty(t, provider.createdPullRequest.Repository.Slug, "the external create completed")
}

func createPullRequestHost() *scopedConnectionHost {
	defaultBranch := "main"
	return &scopedConnectionHost{
		connectionHost: newConnectionHost(),
		tasks: &taskReader{task: &pluginsdk.Task{
			ID: "task-1", WorkspaceID: "workspace-1", Title: "Host task", Description: "Host description",
			Repositories: []pluginsdk.TaskRepository{{RepositoryID: "repository-1", BaseBranch: "main", CheckoutBranch: "feature"}},
		}},
		repositories: &repositoryReader{repositories: []pluginsdk.Repository{{
			ID: "repository-1", WorkspaceID: "workspace-1", Name: "repo", SourceType: "provider", ProviderID: "bitbucket",
			ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-uuid",
			RemoteURL: "https://bitbucket.org/workspace/repo.git", DefaultBranch: &defaultBranch,
		}}},
	}
}

func TestWorkflows_CreatePullRequestRejectsManualRepositoryWithBitbucketFields(t *testing.T) {
	host := newTaskHost()
	host.tasks.task = &pluginsdk.Task{
		ID: "task-1", WorkspaceID: "workspace-1", Title: "Host task",
		Repositories: []pluginsdk.TaskRepository{{RepositoryID: "repository-1", BaseBranch: "main", CheckoutBranch: "feature"}},
	}
	host.repositories.repositories = []pluginsdk.Repository{{
		ID: "repository-1", WorkspaceID: "workspace-1", Name: "repo", SourceType: "manual", ProviderID: "bitbucket",
		ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-uuid",
		RemoteURL: "https://bitbucket.org/workspace/repo.git",
	}}
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: testPullRequest()}})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "task-1"}, Body: []byte(`{}`),
	})
	require.Error(t, err)
}

func TestWorkflows_LaunchAppliesNamedPreset(t *testing.T) {
	host := newTaskHost()
	pullRequest := testPullRequest()
	pullRequest.Repository.CloneURL = mustURL(t, "https://bitbucket.org/workspace/repo.git")
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: pullRequest}})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42","preset":"review"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, host.tasks.created.Launch)
	require.Equal(t, "Review the Bitbucket pull request and run relevant tests.", *host.tasks.created.Launch.Prompt)
}

func TestWorkflows_TasksLaunchMapsNativeTaskOptionsAndTrustedPullRequestRepository(t *testing.T) {
	ctx := context.Background()
	host := &scopedConnectionHost{connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{}}
	pullRequest := launchablePullRequest(t)
	provider := &workflowProvider{pullRequest: pullRequest}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	response, err := workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{
			"review_key":"workspace/repo#42",
			"launch_id":"launch-123",
			"repository":{"provider_id":"bitbucket","provider_host":"attacker.test","owner_or_project":"evil","provider_repository_id":"evil/repo","name":"repo","clone_url":"https://attacker.test/evil/repo.git"},
			"task":{"title":"Review this safely","description":"Trusted launch","workflow_id":"workflow-1","workflow_step_id":"step-1","agent_profile_id":"agent-1","executor_profile_id":"executor-1","start_agent":true,"plan_mode":true}
		}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"task_id":"created-task","linked":true}`, string(response.Body))
	created := host.tasks.created
	require.Equal(t, "workspace-1", created.WorkspaceID)
	require.Equal(t, "workflow-1", created.WorkflowID)
	require.Equal(t, "step-1", requireStringPointer(t, created.WorkflowStepID))
	require.Equal(t, "Review this safely", created.Title)
	require.Equal(t, "Trusted launch", created.Description)
	require.True(t, created.StartAgent)
	require.Equal(t, "agent-1", requireStringPointer(t, created.Launch.AgentProfileID))
	require.Equal(t, "executor-1", requireStringPointer(t, created.Launch.ExecutorProfileID))
	require.Equal(t, "on", requireStringPointer(t, created.Launch.PlanMode))
	require.Len(t, created.Repositories, 1)
	require.Equal(t, "fork", created.Repositories[0].Remote.OwnerOrProject)
	require.Equal(t, "https://bitbucket.org/fork/repo.git", created.Repositories[0].Remote.CloneURL)
	require.Equal(t, "main", requireStringPointer(t, created.Repositories[0].BaseBranch))
	require.Equal(t, "feature/fork", requireStringPointer(t, created.Repositories[0].CheckoutBranch))
	reservationIdentity, err := pullRequestReservationIdentity(pullRequest)
	require.NoError(t, err)
	require.Equal(t, "manual:"+reservationIdentity+":launch-123", created.Metadata["reservation"])
	links, err := workflows.links.List(ctx, "created-task")
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, pullRequest.Key(), links[0].Key)
}

func TestTaskLaunchReservationScopesRetriesToOneDialogLaunch(t *testing.T) {
	first, err := taskLaunchReservation("workspace/repo#42", "launch-one")
	require.NoError(t, err)
	retry, err := taskLaunchReservation("workspace/repo#42", "launch-one")
	require.NoError(t, err)
	second, err := taskLaunchReservation("workspace/repo#42", "launch-two")
	require.NoError(t, err)

	require.Equal(t, first, retry)
	require.NotEqual(t, first, second)
	_, err = taskLaunchReservation("workspace/repo#42", "attacker:value")
	require.ErrorContains(t, err, "launch_id is invalid")
}

func TestPullRequestReservationIdentityChangesAfterRepositoryRecreation(t *testing.T) {
	pullRequest := testPullRequest()
	first, err := pullRequestReservationIdentity(pullRequest)
	require.NoError(t, err)
	pullRequest.Repository.ID = "recreated-repository-uuid"
	second, err := pullRequestReservationIdentity(pullRequest)
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestWorkflows_TasksLaunchCannotRetargetRecreatedRepository(t *testing.T) {
	pullRequest := launchablePullRequest(t)
	pullRequest.Repository.ID = "repo-uuid-new"
	provider := &workflowProvider{pullRequest: pullRequest}
	host := &scopedConnectionHost{
		connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{},
	}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid-old","number":42,"pull_request_id":"42"}`),
	})

	require.ErrorContains(t, err, "repository is unavailable")
	require.Zero(t, host.tasks.createCalls)
}

func TestWorkflows_TasksLaunchRejectsConflictingProviderPullRequestID(t *testing.T) {
	pullRequest := launchablePullRequest(t)
	provider := &workflowProvider{pullRequest: pullRequest}
	host := &scopedConnectionHost{
		connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{},
	}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42","provider_scope":"https://bitbucket.org","repository_id":"repo-uuid","number":42,"pull_request_id":"99"}`),
	})

	require.ErrorContains(t, err, "pull request identity is inconsistent")
	require.Zero(t, provider.getPullRequestCalls)
	require.Zero(t, host.tasks.createCalls)
}

func TestWorkflows_TasksLaunchUsesSafeDefaults(t *testing.T) {
	host := &scopedConnectionHost{connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{}}
	pullRequest := launchablePullRequest(t)
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: pullRequest}})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42"}`),
	})

	require.NoError(t, err)
	require.Equal(t, "Bitbucket PR #42: Fix auth", host.tasks.created.Title)
	require.Equal(t, pullRequest.URL, host.tasks.created.Description)
	require.Empty(t, host.tasks.created.WorkflowID)
	require.Nil(t, host.tasks.created.WorkflowStepID)
	require.False(t, host.tasks.created.StartAgent)
	require.Nil(t, host.tasks.created.Launch)
}

func TestWorkflows_TasksLaunchRejectsUnknownNestedTaskFieldsBeforeProviderLookup(t *testing.T) {
	host := &scopedConnectionHost{connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{}}
	provider := &workflowProvider{pullRequest: launchablePullRequest(t)}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42","task":{"title":"x","repositories":[{"clone_url":"https://attacker.test/repo.git"}]}}`),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
	require.Zero(t, provider.getPullRequestCalls)
	require.Zero(t, host.tasks.createCalls)
}

func TestWorkflows_TasksLaunchAssociationFailureReturnsCreatedTaskWithoutRetryableError(t *testing.T) {
	host := &scopedConnectionHost{connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{}}
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: launchablePullRequest(t)}})
	require.NoError(t, err)
	host.setStateErr = errors.New("state backend leaked detail")

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42"}`),
	})

	require.NoError(t, err, "task creation succeeded, so association failure must not invite a duplicate retry")
	require.JSONEq(t, `{"task_id":"created-task","linked":false,"association_error":"task association could not be saved"}`, string(response.Body))
	require.NotContains(t, string(response.Body), "leaked detail")
	require.Equal(t, 1, host.tasks.createCalls)
}

func TestWorkflows_TasksLaunchReusesReservedTaskAndRepairsAssociation(t *testing.T) {
	ctx := context.Background()
	pullRequest := launchablePullRequest(t)
	reservationIdentity, err := pullRequestReservationIdentity(pullRequest)
	require.NoError(t, err)
	tasks := &taskReader{listed: []pluginsdk.Task{{
		ID: "existing-task", Metadata: map[string]any{sourceMetadataKey: map[string]any{"reservation": "manual:" + reservationIdentity}},
	}}}
	host := &scopedConnectionHost{connectionHost: newConnectionHost(), tasks: tasks, repositories: &repositoryReader{}}
	workflows, err := NewWorkflows(host, staticResolver{provider: &workflowProvider{pullRequest: pullRequest}})
	require.NoError(t, err)

	response, err := workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42"}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"task_id":"existing-task","linked":true}`, string(response.Body))
	response, err = workflows.HandleAction(ctx, &pluginsdk.PluginActionRequest{
		ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"review_key":"workspace/repo#42"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"task_id":"existing-task","linked":true}`, string(response.Body))
	require.Zero(t, tasks.createCalls, "retry must not create another task")
	links, err := workflows.links.List(ctx, "existing-task")
	require.NoError(t, err)
	require.Len(t, links, 1, "association repair must also be idempotent")
}

func launchablePullRequest(t *testing.T) domain.PullRequest {
	t.Helper()
	pullRequest := testPullRequest()
	pullRequest.Repository.CloneURL = mustURL(t, "https://bitbucket.org/workspace/repo.git")
	pullRequest.SourceRepository = domain.Repository{Namespace: "fork", Slug: "repo", CloneURL: mustURL(t, "https://bitbucket.org/fork/repo.git")}
	pullRequest.Source.Name = "feature/fork"
	pullRequest.Destination.Name = "main"
	return pullRequest
}

func requireStringPointer(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	if value == nil {
		return ""
	}
	return *value
}

func TestReviewView_PreservesAdapterFiles(t *testing.T) {
	view := reviewView(domain.Review{
		PullRequest: testPullRequest(),
		Files: []domain.ReviewFile{{
			Path: "internal/auth.go", Status: "modified", Additions: 4, Deletions: 1, Patch: "@@ -1 +1 @@",
		}},
	})

	files, ok := view["files"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, []map[string]any{{
		"path": "internal/auth.go", "status": "modified", "additions": 4, "deletions": 1, "patch": "@@ -1 +1 @@",
	}}, files)
}

func TestReviewView_MarksCurrentViewerAndApproval(t *testing.T) {
	view := reviewView(domain.Review{
		PullRequest: testPullRequest(),
		ViewerID:    "viewer-1",
		Participants: []domain.Participant{
			{ID: "reviewer-1", Name: "Reviewer", Role: "REVIEWER", Approved: false},
			{ID: "VIEWER-1", Name: "Current viewer", Role: "REVIEWER", Approved: true},
		},
	})

	require.Equal(t, true, view["viewer_approved"])
	participants, ok := view["participants"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, true, participants[1]["is_current_user"])
	require.NotContains(t, participants[0], "is_current_user")
}

func TestReviewView_BoundsLazyDetailBelowHostActionLimit(t *testing.T) {
	large := strings.Repeat("x", 16*1024)
	review := domain.Review{PullRequest: testPullRequest(), Diff: strings.Repeat("d", 2*1024*1024)}
	for index := 0; index < 300; index++ {
		review.Files = append(review.Files, domain.ReviewFile{Path: fmt.Sprintf("file-%03d.go", index), Patch: large})
		review.Commits = append(review.Commits, domain.Commit{Hash: fmt.Sprintf("commit-%03d", index), Message: large})
		review.Threads = append(review.Threads, domain.Thread{ID: fmt.Sprint(index), Comments: []domain.Comment{{ID: fmt.Sprint(index), Body: large, Author: "Reviewer"}}})
		review.Statuses = append(review.Statuses, domain.BuildStatus{Key: fmt.Sprintf("build-%03d", index), Name: large, State: "SUCCESSFUL"})
	}

	encoded, err := json.Marshal(reviewView(review))

	require.NoError(t, err)
	require.Less(t, len(encoded), 900*1024, "projected review detail must fit below the host's 1 MiB action cap")
	require.Contains(t, string(encoded), `"truncated_sections"`)
}

func testPullRequest() domain.PullRequest {
	return domain.PullRequest{
		Repository: domain.Repository{ID: "repo-uuid", ProviderScope: "https://bitbucket.org", Namespace: "workspace", Slug: "repo"}, Number: 42, Title: "Fix auth", State: "OPEN", URL: "https://bitbucket.org/workspace/repo/pull-requests/42",
		Capabilities: domain.Capabilities{domain.CapabilityPullRequests: true, domain.CapabilityReview: true, domain.CapabilityMerge: true},
	}
}

type staticResolver struct {
	provider domain.Provider
	err      error
	binding  string
}

type connectionSettingsResolver struct {
	staticResolver
	settings ConnectionSettings
	found    bool
}

func (r *connectionSettingsResolver) Load(context.Context, string) (ConnectionSettings, bool, error) {
	return r.settings, r.found, nil
}

func (r staticResolver) Provider(context.Context, string) (domain.Provider, error) {
	return r.provider, r.err
}

func (r staticResolver) GitCredentialBinding(context.Context, GitCredentialScope) (string, error) {
	if r.binding == "" {
		return "", ErrCredentialUnavailable
	}
	return r.binding, nil
}

type workflowProvider struct {
	pullRequest              domain.PullRequest
	capabilities             domain.Capabilities
	credentialErr            error
	healthErr                error
	healthCalls              int
	actions                  int
	createdPullRequest       domain.CreatePullRequestInput
	inspectedRepositoryURL   string
	inspectedPullRequestURL  string
	repositoryInspectErr     error
	pullRequestInspectErr    error
	getPullRequestCalls      int
	searchQueries            []domain.PullRequestQuery
	searchErr                error
	pullRequestsByRepository map[string][]domain.PullRequest
	pullRequestPages         map[string]map[string]domain.PullRequestPage
	repositories             []domain.Repository
	repositoryPages          map[string]domain.RepositoryPage
	repositoryPageCursors    []string
	repositoryQueries        []string
	listRepositoryLimit      int
	reviewErr                error
	reviewProjections        []domain.ReviewProjection
}

func (p *workflowProvider) Capabilities() domain.Capabilities {
	if p.capabilities != nil {
		return p.capabilities
	}
	return domain.Capabilities{domain.CapabilityPullRequests: true, domain.CapabilityBranches: true, domain.CapabilityReview: true, domain.CapabilityMerge: true}
}

func (p *workflowProvider) ListRepositories(_ context.Context, _ string, limit int) ([]domain.Repository, error) {
	p.listRepositoryLimit = limit
	if p.repositories != nil {
		return testRepositoriesWithIdentity(p.repositories), nil
	}
	return []domain.Repository{testRepositoryWithIdentity(p.pullRequest.Repository)}, nil
}
func (p *workflowProvider) ListRepositoriesPage(_ context.Context, search string, query domain.RepositoryQuery) (domain.RepositoryPage, error) {
	p.listRepositoryLimit = query.Limit
	p.repositoryQueries = append(p.repositoryQueries, search)
	p.repositoryPageCursors = append(p.repositoryPageCursors, query.Cursor)
	if p.repositoryPages != nil {
		page, found := p.repositoryPages[query.Cursor]
		if !found {
			return domain.RepositoryPage{}, fmt.Errorf("unexpected repository cursor %q", query.Cursor)
		}
		page.Repositories = testRepositoriesWithIdentity(page.Repositories)
		return page, nil
	}
	if p.repositories != nil {
		return domain.RepositoryPage{Repositories: testRepositoriesWithIdentity(p.repositories)}, nil
	}
	return domain.RepositoryPage{Repositories: []domain.Repository{testRepositoryWithIdentity(p.pullRequest.Repository)}}, nil
}
func (p *workflowProvider) InspectRepositoryURL(raw string) (domain.Repository, error) {
	p.inspectedRepositoryURL = raw
	if p.repositoryInspectErr != nil {
		return domain.Repository{}, p.repositoryInspectErr
	}
	return p.pullRequest.Repository, nil
}
func (p *workflowProvider) InspectPullRequestURL(raw string) (domain.PullRequestLocator, error) {
	p.inspectedPullRequestURL = raw
	if p.pullRequestInspectErr != nil {
		return domain.PullRequestLocator{}, p.pullRequestInspectErr
	}
	return domain.PullRequestLocator{Repository: p.pullRequest.Repository, Number: p.pullRequest.Number}, nil
}
func (*workflowProvider) ListBranches(context.Context, domain.Repository) ([]domain.Branch, error) {
	return nil, nil
}
func (p *workflowProvider) SearchPullRequests(_ context.Context, query domain.PullRequestQuery) ([]domain.PullRequest, error) {
	p.searchQueries = append(p.searchQueries, query)
	if p.searchErr != nil {
		return nil, p.searchErr
	}
	if p.pullRequestsByRepository != nil {
		return testPullRequestsWithIdentity(p.pullRequestsByRepository[query.Repository.Namespace+"/"+query.Repository.Slug]), nil
	}
	return []domain.PullRequest{testPullRequestWithIdentity(p.pullRequest)}, nil
}
func (p *workflowProvider) SearchPullRequestsPage(ctx context.Context, query domain.PullRequestQuery) (domain.PullRequestPage, error) {
	if p.pullRequestPages != nil {
		p.searchQueries = append(p.searchQueries, query)
		pages := p.pullRequestPages[query.Repository.Namespace+"/"+query.Repository.Slug]
		page, found := pages[query.Cursor]
		if !found {
			return domain.PullRequestPage{}, fmt.Errorf("unexpected pull request cursor %q", query.Cursor)
		}
		page.PullRequests = testPullRequestsWithIdentity(page.PullRequests)
		return page, nil
	}
	pullRequests, err := p.SearchPullRequests(ctx, query)
	return domain.PullRequestPage{PullRequests: pullRequests}, err
}
func (p *workflowProvider) GetPullRequest(_ context.Context, repository domain.Repository, number int) (domain.PullRequest, error) {
	p.getPullRequestCalls++
	if p.pullRequestsByRepository != nil {
		for _, pullRequest := range p.pullRequestsByRepository[repository.Namespace+"/"+repository.Slug] {
			if pullRequest.Number == number {
				return testPullRequestWithIdentity(pullRequest), nil
			}
		}
		return domain.PullRequest{}, errors.New("not found")
	}
	if repository.Namespace != p.pullRequest.Repository.Namespace || repository.Slug != p.pullRequest.Repository.Slug || number != p.pullRequest.Number {
		return domain.PullRequest{}, errors.New("not found")
	}
	return testPullRequestWithIdentity(p.pullRequest), nil
}
func (p *workflowProvider) CreatePullRequest(_ context.Context, input domain.CreatePullRequestInput) (domain.PullRequest, error) {
	p.createdPullRequest = input
	return testPullRequestWithIdentity(p.pullRequest), nil
}
func (p *workflowProvider) GetReview(context.Context, domain.Repository, int) (domain.Review, error) {
	if p.reviewErr != nil {
		return domain.Review{}, p.reviewErr
	}
	return domain.Review{PullRequest: testPullRequestWithIdentity(p.pullRequest)}, nil
}
func (p *workflowProvider) GetReviewProjected(ctx context.Context, repository domain.Repository, number int, projection domain.ReviewProjection) (domain.Review, error) {
	p.reviewProjections = append(p.reviewProjections, projection)
	return p.GetReview(ctx, repository, number)
}
func (p *workflowProvider) ApplyReviewAction(context.Context, domain.PullRequest, domain.ReviewAction) (domain.PullRequest, error) {
	p.actions++
	return testPullRequestWithIdentity(p.pullRequest), nil
}

func testRepositoryWithIdentity(repository domain.Repository) domain.Repository {
	if repository.ID == "" {
		repository.ID = "test:" + repository.Namespace + "/" + repository.Slug
	}
	if repository.ProviderScope == "" {
		repository.ProviderScope = "https://bitbucket.org"
	}
	return repository
}

func testRepositoriesWithIdentity(repositories []domain.Repository) []domain.Repository {
	result := append([]domain.Repository(nil), repositories...)
	for index := range result {
		result[index] = testRepositoryWithIdentity(result[index])
	}
	return result
}

func testPullRequestWithIdentity(pullRequest domain.PullRequest) domain.PullRequest {
	pullRequest.Repository = testRepositoryWithIdentity(pullRequest.Repository)
	if pullRequest.SourceRepository.Namespace != "" {
		pullRequest.SourceRepository = testRepositoryWithIdentity(pullRequest.SourceRepository)
	}
	return pullRequest
}

func testPullRequestsWithIdentity(pullRequests []domain.PullRequest) []domain.PullRequest {
	result := append([]domain.PullRequest(nil), pullRequests...)
	for index := range result {
		result[index] = testPullRequestWithIdentity(result[index])
	}
	return result
}
func (p *workflowProvider) Health(context.Context) error {
	p.healthCalls++
	return p.healthErr
}
func (p *workflowProvider) ResolveGitCredential(context.Context) (domain.GitCredential, error) {
	return domain.GitCredential{}, p.credentialErr
}
