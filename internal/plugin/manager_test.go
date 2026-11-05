package plugin

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kdlbs/kandev/pluginsdk"
	"github.com/stretchr/testify/require"
	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"
)

func TestManager_PollsWorkspaceAndEmitsOnlySafeHealthEvent(t *testing.T) {
	host := &managerHost{connectionHost: newConnectionHost(), workspaces: []pluginsdk.Workspace{{ID: "workspace-1"}}}
	provider := &workflowProvider{healthErr: errors.New("remote rejected secret-token")}
	workflows, err := NewWorkflows(host, staticResolver{provider: provider})
	require.NoError(t, err)
	manager, err := NewManager(host, staticResolver{provider: provider}, workflows.watches)
	require.NoError(t, err)

	err = manager.PollAll(context.Background())
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-token")
	require.Equal(t, []string{"health.unavailable"}, host.events)
}

func TestManager_BackoffIsScopedToTheFailingWorkspace(t *testing.T) {
	host := &managerHost{
		connectionHost: newConnectionHost(),
		workspaces:     []pluginsdk.Workspace{{ID: "broken"}, {ID: "healthy"}},
	}
	broken := &workflowProvider{healthErr: errors.New("unavailable")}
	healthy := &workflowProvider{}
	resolver := workspaceProviderResolver{providers: map[string]*workflowProvider{
		"broken": broken, "healthy": healthy,
	}}
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	manager, err := NewManager(host, resolver, workflows.watches)
	require.NoError(t, err)
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	manager.schedule = domain.NewHealthSchedule(func() float64 { return 0.5 })

	_ = manager.pollScheduled(context.Background())
	now = now.Add(90 * time.Second)
	_ = manager.pollScheduled(context.Background())

	require.Equal(t, 1, broken.healthCalls, "failed workspace must observe its own backoff")
	require.Equal(t, 2, healthy.healthCalls, "healthy workspace must keep its 90-second cadence")
}

func TestManager_WatchRateLimitControlsWorkspaceBackoff(t *testing.T) {
	host := &managerHost{connectionHost: newConnectionHost(), workspaces: []pluginsdk.Workspace{{ID: "workspace-1"}}}
	provider := &workflowProvider{
		pullRequest: testPullRequest(),
		searchErr:   &domain.ProviderHTTPError{Status: http.StatusTooManyRequests, RetryAfter: "600"},
	}
	resolver := staticResolver{provider: provider}
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	_, err = workflows.watches.Create(context.Background(), watches.Watch{
		ID: "watch-1", WorkspaceID: "workspace-1", Status: watches.StatusRunning,
	})
	require.NoError(t, err)
	manager, err := NewManager(host, resolver, workflows.watches)
	require.NoError(t, err)
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	manager.schedule = domain.NewHealthSchedule(func() float64 { return 0.5 })

	err = manager.pollScheduled(context.Background())
	require.Error(t, err)
	require.Len(t, provider.searchQueries, 1)
	now = now.Add(3 * time.Minute)
	_ = manager.pollScheduled(context.Background())
	require.Len(t, provider.searchQueries, 1, "Retry-After must suppress the healthy 90-second cadence")
	now = now.Add(7 * time.Minute)
	_ = manager.pollScheduled(context.Background())
	require.Len(t, provider.searchQueries, 2)
}

type workspaceProviderResolver struct {
	providers map[string]*workflowProvider
}

func (r workspaceProviderResolver) Provider(_ context.Context, workspaceID string) (domain.Provider, error) {
	provider := r.providers[workspaceID]
	if provider == nil {
		return nil, errors.New("provider unavailable")
	}
	return provider, nil
}

type managerHost struct {
	*connectionHost
	workspaces []pluginsdk.Workspace
	events     []string
}

func (h *managerHost) Workspaces() pluginsdk.WorkspaceReader {
	return managerWorkspaceReader{workspaces: h.workspaces}
}
func (h *managerHost) EmitEvent(_ context.Context, name string, _ map[string]any) error {
	h.events = append(h.events, name)
	return nil
}

type managerWorkspaceReader struct{ workspaces []pluginsdk.Workspace }

func (r managerWorkspaceReader) List(context.Context, pluginsdk.Page) ([]pluginsdk.Workspace, *pluginsdk.PageInfo, error) {
	return r.workspaces, nil, nil
}
