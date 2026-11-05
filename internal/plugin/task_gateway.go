package plugin

import (
	"context"
	"fmt"
	"strings"

	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

const sourceMetadataKey = "plugin:kandev-plugin-bitbucket"

// TaskGateway creates and discovers watch-owned tasks through the host's
// provenance-safe data API. It never receives a credential.
type TaskGateway struct{ host pluginsdk.Host }

func NewTaskGateway(host pluginsdk.Host) (*TaskGateway, error) {
	if host == nil {
		return nil, fmt.Errorf("task host is required")
	}
	return &TaskGateway{host: host}, nil
}

func (g *TaskGateway) FindByReservation(ctx context.Context, workspaceID, reservation string) (string, bool, error) {
	if workspaceID == "" || reservation == "" {
		return "", false, fmt.Errorf("workspace id and reservation are required")
	}
	page := pluginsdk.Page{Limit: 100}
	for {
		tasks, info, err := g.host.Tasks().List(ctx, pluginsdk.TaskFilter{WorkspaceIDs: []string{workspaceID}}, page)
		if err != nil {
			return "", false, fmt.Errorf("list workspace tasks: %w", err)
		}
		for _, task := range tasks {
			if taskReservation(task.Metadata) == reservation {
				return task.ID, true, nil
			}
		}
		if info == nil || !info.HasMore || info.NextCursor == "" {
			return "", false, nil
		}
		page.Cursor = info.NextCursor
	}
}

func (g *TaskGateway) Create(ctx context.Context, creation watches.Creation) (string, error) {
	remote, err := remoteRepository(creation.PullRequest)
	if err != nil {
		return "", err
	}
	if creation.ReservationToken == "" {
		return "", fmt.Errorf("task reservation token is required")
	}
	launch := launchOptions(creation.Watch.Launch)
	number := creation.PullRequest.Number
	title := strings.TrimSpace(creation.PullRequest.Title)
	if title == "" {
		title = fmt.Sprintf("Bitbucket pull request #%d", number)
	} else {
		title = fmt.Sprintf("Bitbucket PR #%d: %s", number, title)
	}
	created, err := g.host.Tasks().Create(ctx, pluginsdk.CreateTaskInput{
		WorkspaceID: creation.WorkspaceID,
		WorkflowID:  creation.Watch.Launch.WorkflowID,
		Title:       title,
		Description: creation.PullRequest.URL,
		StartAgent:  creation.Watch.Launch.StartAgent,
		Repositories: []pluginsdk.PluginTaskRepository{{
			Remote:         remote,
			BaseBranch:     stringPointer(creation.PullRequest.Repository.BaseBranch),
			CheckoutBranch: stringPointer(creation.PullRequest.Repository.HeadBranch),
		}},
		Launch: launch,
		Metadata: map[string]any{
			"watch_id":         creation.Watch.ID,
			"pull_request_key": creation.PullRequest.Key,
			"reservation":      creation.ReservationToken,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create watch task: %w", err)
	}
	if created == nil || created.ID == "" {
		return "", fmt.Errorf("create watch task returned no task id")
	}
	return created.ID, nil
}

func (g *TaskGateway) PreviewOwned(ctx context.Context, rootTaskID string) ([]string, error) {
	manager, ok := pluginsdk.PluginOwnedTaskTrees(g.host)
	if !ok {
		return nil, fmt.Errorf("plugin-owned task manager is unavailable")
	}
	tasks, err := manager.Preview(ctx, rootTaskID)
	if err != nil {
		return nil, fmt.Errorf("preview plugin-owned task tree: %w", err)
	}
	return taskIDs(tasks), nil
}

func (g *TaskGateway) DeleteOwned(ctx context.Context, rootTaskID string) ([]string, error) {
	manager, ok := pluginsdk.PluginOwnedTaskTrees(g.host)
	if !ok {
		return nil, fmt.Errorf("plugin-owned task manager is unavailable")
	}
	deleted, err := manager.Delete(ctx, rootTaskID)
	if err != nil {
		return deleted, fmt.Errorf("delete plugin-owned task tree: %w", err)
	}
	return deleted, nil
}

func taskReservation(metadata map[string]any) string {
	pluginMetadata, ok := metadata[sourceMetadataKey].(map[string]any)
	if !ok {
		return ""
	}
	reservation, _ := pluginMetadata["reservation"].(string)
	return reservation
}

func remoteRepository(pr watches.PullRequest) (*pluginsdk.RemoteRepositoryDescriptor, error) {
	repository := pr.Repository
	if repository.ProviderID == "" || repository.ProviderHost == "" || repository.ProviderScope == "" || repository.OwnerOrProject == "" || repository.ProviderRepositoryID == "" || repository.Name == "" || repository.CloneURL == "" {
		return nil, fmt.Errorf("pull request %q has incomplete repository descriptor", pr.Key)
	}
	return &pluginsdk.RemoteRepositoryDescriptor{
		ProviderID:           repository.ProviderID,
		ProviderHost:         repository.ProviderHost,
		ProviderScope:        repository.ProviderScope,
		OwnerOrProject:       repository.OwnerOrProject,
		ProviderRepositoryID: repository.ProviderRepositoryID,
		Name:                 repository.Name,
		CloneURL:             repository.CloneURL,
		DefaultBranch:        stringPointer(repository.DefaultBranch),
		BaseBranch:           stringPointer(repository.BaseBranch),
		HeadBranch:           stringPointer(repository.HeadBranch),
	}, nil
}

func launchOptions(launch watches.Launch) *pluginsdk.PluginTaskLaunchOptions {
	options := &pluginsdk.PluginTaskLaunchOptions{
		AgentProfileID:    stringPointer(launch.AgentProfileID),
		ExecutorProfileID: stringPointer(launch.ExecutorProfileID),
		Prompt:            stringPointer(launch.Prompt),
	}
	if options.AgentProfileID == nil && options.ExecutorProfileID == nil && options.Prompt == nil {
		return nil
	}
	return options
}

func taskIDs(tasks []pluginsdk.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task.ID != "" {
			ids = append(ids, task.ID)
		}
	}
	return ids
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func int64Pointer(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}
