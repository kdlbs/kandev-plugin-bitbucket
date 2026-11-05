package plugin

import (
	"context"
	"fmt"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

func (w *Workflows) launchWorkspaceTask(ctx context.Context, workspaceID string, pullRequest domain.PullRequest, input launchTaskInput) (*pluginsdk.PluginActionResponse, error) {
	taskInput, prompt, err := nativeTaskOptions(input)
	if err != nil {
		return nil, err
	}
	reservationIdentity, err := pullRequestReservationIdentity(pullRequest)
	if err != nil {
		return nil, err
	}
	reservation, err := taskLaunchReservation(reservationIdentity, input.LaunchID)
	if err != nil {
		return nil, err
	}
	taskID, found, err := w.tasks.FindByReservation(ctx, workspaceID, reservation)
	if err != nil {
		return nil, err
	}
	if !found {
		taskID, err = w.createWorkspaceTask(ctx, workspaceID, pullRequest, taskInput, prompt, reservation)
		if err != nil {
			return nil, err
		}
	}
	response := map[string]any{"task_id": taskID}
	link, associationErr := w.linkForPullRequest(ctx, workspaceID, pullRequest)
	if associationErr == nil {
		_, associationErr = w.links.Link(ctx, taskID, link)
	}
	response["linked"] = associationErr == nil
	if associationErr != nil {
		response["association_error"] = "task association could not be saved"
	}
	return actionResponse(response)
}

func taskLaunchReservation(pullRequestKey, launchID string) (string, error) {
	reservation := "manual:" + pullRequestKey
	launchID = strings.TrimSpace(launchID)
	if launchID == "" {
		return reservation, nil
	}
	if len(launchID) > 128 {
		return "", invalidActionError("launch_id is invalid")
	}
	for _, char := range launchID {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' {
			return "", invalidActionError("launch_id is invalid")
		}
	}
	return reservation + ":" + launchID, nil
}

func pullRequestReservationIdentity(pullRequest domain.PullRequest) (string, error) {
	identity, complete := (domain.PullRequestIdentity{
		ProviderID: "bitbucket", ProviderScope: pullRequest.Repository.ProviderScope,
		RepositoryID: pullRequest.Repository.ID, Number: int64(pullRequest.Number),
	}).StorageKey()
	if !complete {
		return "", conflictActionError("pull request immutable identity is unavailable")
	}
	return identity, nil
}

func nativeTaskOptions(input launchTaskInput) (nativeTaskInput, string, error) {
	if input.Task != nil {
		if input.Preset != "" || input.Launch != (watches.Launch{}) {
			return nativeTaskInput{}, "", invalidActionError("native and legacy task settings cannot be combined")
		}
		return *input.Task, "", nil
	}
	if err := applyLaunchPreset(&input.Launch, input.Preset); err != nil {
		return nativeTaskInput{}, "", err
	}
	return nativeTaskInput{
		WorkflowID:        input.Launch.WorkflowID,
		WorkflowStepID:    input.Launch.WorkflowStepID,
		AgentProfileID:    input.Launch.AgentProfileID,
		ExecutorProfileID: input.Launch.ExecutorProfileID,
		StartAgent:        input.Launch.StartAgent,
	}, input.Launch.Prompt, nil
}

func (w *Workflows) createWorkspaceTask(ctx context.Context, workspaceID string, pullRequest domain.PullRequest, task nativeTaskInput, prompt, reservation string) (string, error) {
	remote, err := remoteRepository(watchPullRequest(pullRequest))
	if err != nil {
		return "", err
	}
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = strings.TrimSpace(pullRequest.Title)
		if title == "" {
			title = fmt.Sprintf("Bitbucket pull request #%d", pullRequest.Number)
		} else {
			title = fmt.Sprintf("Bitbucket PR #%d: %s", pullRequest.Number, title)
		}
	}
	description := strings.TrimSpace(task.Description)
	if description == "" {
		description = pullRequest.URL
	}
	launch := &pluginsdk.PluginTaskLaunchOptions{
		AgentProfileID:    stringPointer(strings.TrimSpace(task.AgentProfileID)),
		ExecutorProfileID: stringPointer(strings.TrimSpace(task.ExecutorProfileID)),
		Prompt:            stringPointer(strings.TrimSpace(prompt)),
	}
	if task.PlanMode {
		launch.PlanMode = stringPointer("on")
	}
	if launch.AgentProfileID == nil && launch.ExecutorProfileID == nil && launch.Prompt == nil && launch.PlanMode == nil {
		launch = nil
	}
	created, err := w.host.Tasks().Create(ctx, pluginsdk.CreateTaskInput{
		WorkspaceID:    workspaceID,
		WorkflowID:     strings.TrimSpace(task.WorkflowID),
		WorkflowStepID: stringPointer(strings.TrimSpace(task.WorkflowStepID)),
		Title:          title,
		Description:    description,
		StartAgent:     task.StartAgent,
		Repositories: []pluginsdk.PluginTaskRepository{{
			Remote:         remote,
			BaseBranch:     stringPointer(pullRequest.Destination.Name),
			CheckoutBranch: stringPointer(pullRequest.Source.Name),
		}},
		Launch: launch,
		Metadata: map[string]any{
			"pull_request_key": pullRequest.Key(),
			"reservation":      reservation,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create Bitbucket task: %w", err)
	}
	if created == nil || created.ID == "" {
		return "", fmt.Errorf("create Bitbucket task returned no task id")
	}
	return created.ID, nil
}
