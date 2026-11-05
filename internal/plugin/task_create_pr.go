package plugin

import (
	"context"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

// createPullRequestFromTask derives all repository authority from the
// host-verified task and repository records. Browser input may only override
// non-authoritative PR presentation fields.
func (w *Workflows) createPullRequestFromTask(ctx context.Context, action pluginsdk.VerifiedActionContext, input taskCreatePullRequestInput) (domain.Provider, domain.Repository, string, string, string, string, error) {
	task, err := w.host.Tasks().Get(ctx, action.TaskID)
	if err != nil {
		return nil, domain.Repository{}, "", "", "", "", unavailableActionError("load verified task: %v", err)
	}
	if task == nil || task.ID != action.TaskID || task.WorkspaceID != action.WorkspaceID {
		return nil, domain.Repository{}, "", "", "", "", forbiddenActionError("verified task is unavailable")
	}
	verifiedTask := taskWithVerifiedHeadBranch(*task, action)
	repositories, err := taskBitbucketRepositories(ctx, w.host, verifiedTask)
	if err != nil {
		return nil, domain.Repository{}, "", "", "", "", err
	}
	candidate, err := selectTaskBitbucketRepository(repositories, action.RepositoryID)
	if err != nil {
		return nil, domain.Repository{}, "", "", "", "", err
	}
	remote := watches.RemoteRepository{
		ProviderID: "bitbucket", ProviderHost: candidate.repository.ProviderHost,
		ProviderScope:  candidate.repository.ProviderScope,
		OwnerOrProject: candidate.repository.OwnerOrProject, ProviderRepositoryID: candidate.repository.ProviderRepositoryID,
		Name: providerRepositoryName(candidate.repository), CloneURL: candidate.repository.RemoteURL,
		DefaultBranch: stringValue(candidate.repository.DefaultBranch), BaseBranch: candidate.taskRepository.BaseBranch,
		HeadBranch: candidate.taskRepository.CheckoutBranch,
	}
	repository, err := domainRepository(remote)
	if err != nil {
		return nil, domain.Repository{}, "", "", "", "", conflictActionError("task Bitbucket repository is invalid")
	}
	provider, err := w.provider(ctx, action.WorkspaceID)
	if err != nil {
		return nil, domain.Repository{}, "", "", "", "", err
	}
	destination := strings.TrimSpace(input.Destination)
	if destination == "" {
		destination = strings.TrimSpace(candidate.taskRepository.BaseBranch)
	}
	if destination == "" {
		destination = stringValue(candidate.repository.DefaultBranch)
	}
	source := branchName(candidate.taskRepository.CheckoutBranch)
	destination = branchName(destination)
	if source == "" || destination == "" {
		return nil, domain.Repository{}, "", "", "", "", conflictActionError("task checkout and destination branches are required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = strings.TrimSpace(task.Title)
	}
	if title == "" {
		return nil, domain.Repository{}, "", "", "", "", invalidActionError("pull request title is required")
	}
	description := input.Description
	if description == "" {
		description = task.Description
	}
	return provider, repository, source, destination, title, description, nil
}

func taskWithVerifiedHeadBranch(task pluginsdk.Task, action pluginsdk.VerifiedActionContext) pluginsdk.Task {
	repositoryID := strings.TrimSpace(action.RepositoryID)
	headBranch := strings.TrimSpace(action.HeadBranch)
	if repositoryID == "" || headBranch == "" {
		return task
	}
	task.Repositories = append([]pluginsdk.TaskRepository(nil), task.Repositories...)
	for i := range task.Repositories {
		if task.Repositories[i].RepositoryID == repositoryID {
			task.Repositories[i].CheckoutBranch = headBranch
			break
		}
	}
	return task
}

func selectTaskBitbucketRepository(repositories []taskRepositoryCandidate, repositoryID string) (taskRepositoryCandidate, error) {
	verifiedRepositoryID := strings.TrimSpace(repositoryID)
	if verifiedRepositoryID == "" {
		if len(repositories) != 1 {
			return taskRepositoryCandidate{}, invalidActionError("task must have exactly one Bitbucket repository with a checkout branch")
		}
		return repositories[0], nil
	}
	for _, candidate := range repositories {
		if candidate.repository.ID == verifiedRepositoryID {
			return candidate, nil
		}
	}
	return taskRepositoryCandidate{}, forbiddenActionError("verified repository is not an attached Bitbucket checkout")
}

type taskRepositoryCandidate struct {
	taskRepository pluginsdk.TaskRepository
	repository     pluginsdk.Repository
}

func taskBitbucketRepositories(ctx context.Context, host pluginsdk.Host, task pluginsdk.Task) ([]taskRepositoryCandidate, error) {
	if len(task.Repositories) == 0 {
		return nil, conflictActionError("task has no repository checkout")
	}
	repositoryIDs := make(map[string]pluginsdk.TaskRepository, len(task.Repositories))
	for _, taskRepository := range task.Repositories {
		if taskRepository.RepositoryID == "" || strings.TrimSpace(taskRepository.CheckoutBranch) == "" {
			continue
		}
		if _, found := repositoryIDs[taskRepository.RepositoryID]; found {
			return nil, conflictActionError("task repository checkout is ambiguous")
		}
		repositoryIDs[taskRepository.RepositoryID] = taskRepository
	}
	if len(repositoryIDs) == 0 {
		return nil, conflictActionError("task has no repository checkout branch")
	}
	result := make([]taskRepositoryCandidate, 0, len(repositoryIDs))
	page := pluginsdk.Page{Limit: 100}
	for {
		repositories, info, err := host.Repositories().List(ctx, task.WorkspaceID, page)
		if err != nil {
			return nil, unavailableActionError("load task repositories: %v", err)
		}
		for _, repository := range repositories {
			taskRepository, wanted := repositoryIDs[repository.ID]
			if !wanted {
				continue
			}
			if repository.SourceType != "provider" || repository.ProviderID != "bitbucket" {
				continue
			}
			if repository.ProviderHost == "" || repository.ProviderScope == "" || repository.OwnerOrProject == "" || repository.ProviderRepositoryID == "" || providerRepositoryName(repository) == "" || repository.RemoteURL == "" {
				return nil, conflictActionError("task Bitbucket repository origin is incomplete")
			}
			result = append(result, taskRepositoryCandidate{taskRepository: taskRepository, repository: repository})
		}
		if info == nil || !info.HasMore || info.NextCursor == "" {
			break
		}
		page.Cursor = info.NextCursor
	}
	return result, nil
}

func providerRepositoryName(repository pluginsdk.Repository) string {
	if name := strings.TrimSpace(repository.ProviderName); name != "" {
		return name
	}
	name := strings.Trim(strings.TrimSpace(repository.Name), "/")
	if separator := strings.LastIndex(name, "/"); separator >= 0 {
		return name[separator+1:]
	}
	return name
}

func branchName(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "refs/heads/")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
