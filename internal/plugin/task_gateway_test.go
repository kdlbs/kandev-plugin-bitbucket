package plugin

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kandev/kandev/pkg/pluginsdk"
	"github.com/stretchr/testify/require"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}

func TestTaskGateway_CreateUsesCredentialFreeRemoteAndDurableReservation(t *testing.T) {
	host := newTaskHost()
	gateway, err := NewTaskGateway(host)
	require.NoError(t, err)
	_, err = gateway.Create(context.Background(), watches.Creation{
		WorkspaceID: "workspace-1",
		Watch:       watches.Watch{ID: "watch-1", Launch: watches.Launch{WorkflowID: "workflow-1", StartAgent: true}},
		PullRequest: watches.PullRequest{
			Key: "repo-1#42", Number: 42, Title: "Fix race", URL: "https://bitbucket.org/workspace/repo/pull-requests/42",
			Repository: watches.RemoteRepository{ProviderID: "bitbucket", ProviderHost: "bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "workspace", ProviderRepositoryID: "repo-1", Name: "repo", CloneURL: "https://bitbucket.org/workspace/repo.git", BaseBranch: "main", HeadBranch: "fix-race"},
		},
		ReservationToken: "reservation-1",
	})
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.org/workspace/repo.git", host.tasks.created.Repositories[0].Remote.CloneURL)
	require.NotContains(t, host.tasks.created.Repositories[0].Remote.CloneURL, "@")
	require.Nil(t, host.tasks.created.Repositories[0].PullRequestNumber, "Bitbucket task launches do not use GitHub PR-number fields")
	require.Equal(t, "reservation-1", host.tasks.created.Metadata["reservation"])
	require.True(t, host.tasks.created.StartAgent)
}

func TestWatchPullRequest_UsesForkSourceDescriptor(t *testing.T) {
	pullRequest := domain.PullRequest{
		Repository:       domain.Repository{ID: "destination-repo-uuid", ProviderScope: "https://bitbucket.org", Namespace: "destination", Slug: "repo"},
		SourceRepository: domain.Repository{ID: "fork-repo-uuid", ProviderScope: "https://bitbucket.org", Namespace: "fork", Slug: "repo", CloneURL: mustURL(t, "https://bitbucket.org/fork/repo.git")},
		Number:           42,
		Source:           domain.Branch{Name: "feature/fork"},
		Destination:      domain.Branch{Name: "main"},
	}

	got := watchPullRequest(pullRequest)
	require.Equal(t, "fork", got.Repository.OwnerOrProject)
	require.Equal(t, "fork-repo-uuid", got.Repository.ProviderRepositoryID)
	require.Equal(t, "destination-repo-uuid", got.RepositoryID, "pull-request identity belongs to the destination repository")
	require.Equal(t, "https://bitbucket.org/fork/repo.git", got.Repository.CloneURL)
	require.Equal(t, "feature/fork", got.Repository.HeadBranch)
	require.Equal(t, "main", got.Repository.BaseBranch)
}

func TestTaskGateway_DeleteOwnedPreservesPartialProgress(t *testing.T) {
	host := newTaskHost()
	host.trees.deleted = []string{"grandchild", "child"}
	host.trees.deleteErr = errors.New("task store unavailable")
	gateway, err := NewTaskGateway(host)
	require.NoError(t, err)

	deletedTaskIDs, err := gateway.DeleteOwned(context.Background(), "root")

	require.Equal(t, []string{"grandchild", "child"}, deletedTaskIDs)
	require.ErrorContains(t, err, "task store unavailable")
}

func TestTaskGateway_FindsOnlyTaskWithPluginNamespacedReservation(t *testing.T) {
	host := newTaskHost()
	host.tasks.listed = []pluginsdk.Task{
		{ID: "other", Metadata: map[string]any{sourceMetadataKey: map[string]any{"reservation": "other"}}},
		{ID: "wanted", Metadata: map[string]any{sourceMetadataKey: map[string]any{"reservation": "reservation-1"}}},
	}
	gateway, err := NewTaskGateway(host)
	require.NoError(t, err)
	taskID, found, err := gateway.FindByReservation(context.Background(), "workspace-1", "reservation-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "wanted", taskID)
}

func TestTaskGateway_DelegatesCleanupToHostOwnershipManager(t *testing.T) {
	host := newTaskHost()
	host.trees.preview = []pluginsdk.Task{{ID: "root"}, {ID: "owned-child"}}
	host.trees.deleted = []string{"owned-child", "root"}
	gateway, err := NewTaskGateway(host)
	require.NoError(t, err)
	preview, err := gateway.PreviewOwned(context.Background(), "root")
	require.NoError(t, err)
	require.Equal(t, []string{"root", "owned-child"}, preview)
	deleted, err := gateway.DeleteOwned(context.Background(), "root")
	require.NoError(t, err)
	require.Equal(t, []string{"owned-child", "root"}, deleted)
	require.Equal(t, []string{"root"}, host.trees.previewRoots)
	require.Equal(t, []string{"root"}, host.trees.deleteRoots)
}

type taskHost struct {
	pluginsdk.UnimplementedHostData
	tasks        *taskReader
	repositories *repositoryReader
	trees        *ownedTrees
}

func newTaskHost() *taskHost {
	return &taskHost{tasks: &taskReader{}, repositories: &repositoryReader{}, trees: &ownedTrees{}}
}

func (*taskHost) GetState(context.Context, string, string, string) (map[string]any, bool, error) {
	return nil, false, nil
}
func (*taskHost) SetState(context.Context, string, string, string, map[string]any) error {
	return nil
}
func (*taskHost) DeleteState(context.Context, string, string, string) error { return nil }
func (*taskHost) ListState(context.Context, string, string) ([]pluginsdk.StateEntry, error) {
	return nil, nil
}
func (*taskHost) GetConfig(context.Context) (map[string]any, error)            { return map[string]any{}, nil }
func (*taskHost) RevealSecret(context.Context, string) (string, error)         { return "", nil }
func (*taskHost) GetSecret(context.Context, string) (string, bool, error)      { return "", false, nil }
func (*taskHost) SetSecret(context.Context, string, string) error              { return nil }
func (*taskHost) DeleteSecret(context.Context, string) error                   { return nil }
func (*taskHost) EmitEvent(context.Context, string, map[string]any) error      { return nil }
func (h *taskHost) Tasks() pluginsdk.TaskReader                                { return h.tasks }
func (h *taskHost) Repositories() pluginsdk.RepositoryReader                   { return h.repositories }
func (h *taskHost) PluginOwnedTaskTrees() pluginsdk.PluginOwnedTaskTreeManager { return h.trees }

type taskReader struct {
	created      pluginsdk.CreateTaskInput
	listed       []pluginsdk.Task
	task         *pluginsdk.Task
	createCalls  int
	createErr    error
	createResult *pluginsdk.Task
}

func (r *taskReader) List(context.Context, pluginsdk.TaskFilter, pluginsdk.Page) ([]pluginsdk.Task, *pluginsdk.PageInfo, error) {
	return r.listed, nil, nil
}

func (r *taskReader) Get(context.Context, string) (*pluginsdk.Task, error) { return r.task, nil }

func (r *taskReader) Create(_ context.Context, input pluginsdk.CreateTaskInput) (*pluginsdk.Task, error) {
	r.createCalls++
	r.created = input
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createResult != nil {
		return r.createResult, nil
	}
	return &pluginsdk.Task{ID: "created-task", CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}

func (*taskReader) Update(context.Context, pluginsdk.UpdateTaskInput) (*pluginsdk.Task, error) {
	return nil, nil
}

func (*taskReader) Move(context.Context, pluginsdk.MoveTaskInput) (*pluginsdk.MoveTaskOutcome, error) {
	return nil, errors.New("taskReader.Move is not implemented")
}

type repositoryReader struct{ repositories []pluginsdk.Repository }

func (r *repositoryReader) List(context.Context, string, pluginsdk.Page) ([]pluginsdk.Repository, *pluginsdk.PageInfo, error) {
	return r.repositories, nil, nil
}

type ownedTrees struct {
	preview      []pluginsdk.Task
	deleted      []string
	previewRoots []string
	deleteRoots  []string
	deleteErr    error
}

func (m *ownedTrees) Preview(_ context.Context, root string) ([]pluginsdk.Task, error) {
	m.previewRoots = append(m.previewRoots, root)
	return m.preview, nil
}

func (m *ownedTrees) Delete(_ context.Context, root string) ([]string, error) {
	m.deleteRoots = append(m.deleteRoots, root)
	return m.deleted, m.deleteErr
}
