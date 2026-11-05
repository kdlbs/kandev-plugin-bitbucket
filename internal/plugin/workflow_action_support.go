package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

func requireCapability(provider domain.Provider, capability domain.Capability) error {
	if !provider.Capabilities().Supports(capability) {
		return pluginsdk.CategorizeActionError(
			pluginsdk.ActionErrorInvalidArgument,
			fmt.Errorf("Bitbucket connection does not support %s", capability),
		)
	}
	return nil
}

func decodeAction(body []byte, target any) error {
	if len(body) == 0 {
		body = []byte("{}")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorInvalidArgument, fmt.Errorf("invalid action body: %w", err))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorInvalidArgument, fmt.Errorf("invalid action body: multiple JSON values"))
		}
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorInvalidArgument, fmt.Errorf("invalid action body: %w", err))
	}
	return nil
}

func actionResponse(value any) (*pluginsdk.PluginActionResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &pluginsdk.PluginActionResponse{Body: body, Headers: map[string]string{"Content-Type": "application/json"}}, nil
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func applyLaunchPreset(launch *watches.Launch, preset string) error {
	if launch == nil {
		return invalidActionError("launch settings are required")
	}
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "", "default":
		return nil
	case "review":
		if launch.Prompt == "" {
			launch.Prompt = "Review the Bitbucket pull request and run relevant tests."
		}
	case "implement":
		if launch.Prompt == "" {
			launch.Prompt = "Implement the Bitbucket pull request changes and run relevant tests."
		}
	default:
		return invalidActionError("unknown Bitbucket launch preset")
	}
	return nil
}
func safeError(err error) string {
	if err == nil {
		return ""
	}
	return "Bitbucket connection health check failed"
}

type listRepositoriesInput struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}
type repositoryInput struct {
	Repository watches.RemoteRepository `json:"repository"`
}
type repositoryInspectInput struct {
	URL string `json:"url"`
}
type searchPullRequestsInput struct {
	Repository watches.RemoteRepository `json:"repository"`
	Query      string                   `json:"query"`
	State      string                   `json:"state"`
	Limit      int                      `json:"limit"`
	Cursor     string                   `json:"cursor"`
}
type queuePullRequestsInput struct {
	Query  string `json:"query"`
	State  string `json:"state"`
	Limit  int    `json:"limit"`
	View   string `json:"view"`
	Cursor string `json:"cursor"`
}
type pullRequestAssociationsInput struct {
	ReviewKeys *[]string `json:"review_keys"`
}
type pullRequestLookup struct {
	ReviewKey     string                   `json:"review_key"`
	ProviderScope string                   `json:"provider_scope"`
	RepositoryID  string                   `json:"repository_id"`
	Repository    watches.RemoteRepository `json:"repository"`
	Number        int                      `json:"number"`
	PullRequestID string                   `json:"pull_request_id"`
	View          string                   `json:"view"`
	Include       []string                 `json:"include"`
}
type taskCreatePullRequestInput struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	Destination        string `json:"destination"`
	CloseSourceOnMerge bool   `json:"close_source_on_merge"`
}
type reviewActionInput struct {
	pullRequestLookup
	Kind            string `json:"kind"`
	Operation       string `json:"operation"`
	Comment         string `json:"comment"`
	ParentCommentID string `json:"parent_comment_id"`
	BuildKey        string `json:"build_key"`
}
type launchTaskInput struct {
	pullRequestLookup
	Launch   watches.Launch   `json:"launch"`
	Preset   string           `json:"preset"`
	Task     *nativeTaskInput `json:"task"`
	LaunchID string           `json:"launch_id"`
}
type nativeTaskInput struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	WorkflowID        string `json:"workflow_id"`
	WorkflowStepID    string `json:"workflow_step_id"`
	AgentProfileID    string `json:"agent_profile_id"`
	ExecutorProfileID string `json:"executor_profile_id"`
	StartAgent        bool   `json:"start_agent"`
	PlanMode          bool   `json:"plan_mode"`
}
type unlinkInput struct {
	ReviewKey     string `json:"review_key"`
	ProviderScope string `json:"provider_scope"`
	RepositoryID  string `json:"repository_id"`
	Number        int64  `json:"number"`
}
type watchIDInput struct {
	WatchID string `json:"watch_id"`
}
type watchFilterInput struct {
	WatchID string         `json:"watch_id"`
	Filter  watches.Filter `json:"filter"`
}
type watchPresetInput struct {
	WatchID string         `json:"watch_id"`
	Preset  watches.Preset `json:"preset"`
}
type watchUpdateInput struct {
	WatchID string         `json:"watch_id"`
	Enabled bool           `json:"enabled"`
	Filter  watches.Filter `json:"filter"`
	Launch  watches.Launch `json:"launch"`
}
type connectionSaveInput struct {
	ConnectionInput
	Probe bool `json:"probe"`
}

func connectionResponse(settings ConnectionSettings, healthy bool, err error, oauthConfigured ...bool) map[string]any {
	state := "connected"
	if !healthy {
		state = "auth_required"
		if err != nil && !isAuthenticationFailure(err) {
			state = "unavailable"
		}
	}
	registrationConfigured := settings.AuthMethod == "oauth" && settings.OAuthClientID != ""
	if len(oauthConfigured) > 0 {
		registrationConfigured = oauthConfigured[0]
	}
	return map[string]any{
		"state": state, "healthy": healthy, "product": settings.Product, "base_url": settings.BaseURL,
		"cloud_workspace": settings.CloudWorkspace, "auth_method": settings.AuthMethod, "auth_identity": settings.AuthIdentity,
		"oauth_client_id": settings.OAuthClientID, "oauth_redirect_url": settings.OAuthRedirectURL,
		"oauth_registration_configured": registrationConfigured,
		"error":                         safeError(err),
	}
}

func isAuthenticationFailure(err error) bool {
	var providerError *domain.ProviderHTTPError
	if errors.As(err, &providerError) {
		return providerError.Status == http.StatusUnauthorized || providerError.Status == http.StatusForbidden
	}
	return errors.Is(err, ErrCredentialUnavailable)
}
