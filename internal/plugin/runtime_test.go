package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
	"github.com/stretchr/testify/require"
)

func TestRuntime_MapsExpectedActionFailuresToDomainResponses(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	runtime := &Runtime{workflows: workflows}

	response, err := runtime.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "repositories.inspect",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"url":42}`),
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, response.Status)
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body, &body))
	require.Equal(t, string(pluginsdk.ActionErrorInvalidArgument), body["code"])
	require.NotContains(t, string(response.Body), "cannot unmarshal")
}

func TestRuntime_MapsLocalWorkflowFailuresToTypedResponses(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	runtime := &Runtime{workflows: workflows}
	tests := []struct {
		name       string
		request    *pluginsdk.PluginActionRequest
		wantStatus int
		wantCode   pluginsdk.ActionErrorCode
	}{
		{
			name:       "unsafe repository URL",
			request:    &pluginsdk.PluginActionRequest{ActionKey: "repositories.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"url":"http://bitbucket.org/workspace/repo"}`)},
			wantStatus: http.StatusBadRequest, wantCode: pluginsdk.ActionErrorInvalidArgument,
		},
		{
			name:       "missing task authorization",
			request:    &pluginsdk.PluginActionRequest{ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{}`)},
			wantStatus: http.StatusForbidden, wantCode: pluginsdk.ActionErrorPermissionDenied,
		},
		{
			name:       "missing pull request",
			request:    &pluginsdk.PluginActionRequest{ActionKey: "pullrequests.inspect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"review_key":"workspace/missing#42"}`)},
			wantStatus: http.StatusNotFound, wantCode: pluginsdk.ActionErrorNotFound,
		},
		{
			name:       "unknown launch preset",
			request:    &pluginsdk.PluginActionRequest{ActionKey: "tasks.launch", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{"review_key":"workspace/repo#42","preset":"unknown"}`)},
			wantStatus: http.StatusBadRequest, wantCode: pluginsdk.ActionErrorInvalidArgument,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, actionErr := runtime.HandleAction(context.Background(), test.request)
			require.NoError(t, actionErr)
			require.Equal(t, test.wantStatus, response.Status)
			var body map[string]any
			require.NoError(t, json.Unmarshal(response.Body, &body))
			require.Equal(t, string(test.wantCode), body["code"])
		})
	}
}

func TestRuntime_MapsUnavailableConnectionCapabilitiesAndMissingVerifiedTask(t *testing.T) {
	provider := &workflowProvider{pullRequest: testPullRequest()}
	plainHost := newConnectionHost()
	plainWorkflows, err := NewWorkflows(plainHost, staticResolver{provider: provider})
	require.NoError(t, err)
	unavailableWorkflows, err := NewWorkflows(
		plainHost,
		staticResolver{err: errors.New("connection state unavailable")},
	)
	require.NoError(t, err)
	missingTaskHost := &scopedConnectionHost{
		connectionHost: newConnectionHost(), tasks: &taskReader{}, repositories: &repositoryReader{},
	}
	missingTaskWorkflows, err := NewWorkflows(missingTaskHost, staticResolver{provider: provider})
	require.NoError(t, err)

	tests := []struct {
		name       string
		runtime    *Runtime
		request    *pluginsdk.PluginActionRequest
		wantStatus int
		wantCode   pluginsdk.ActionErrorCode
	}{
		{
			name: "connection updates unavailable", runtime: &Runtime{workflows: plainWorkflows},
			request:    &pluginsdk.PluginActionRequest{ActionKey: "connection.disconnect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}},
			wantStatus: http.StatusServiceUnavailable, wantCode: pluginsdk.ActionErrorUnavailable,
		},
		{
			name: "OAuth coordinator unavailable", runtime: &Runtime{workflows: plainWorkflows},
			request:    &pluginsdk.PluginActionRequest{ActionKey: "oauth.start", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}},
			wantStatus: http.StatusConflict, wantCode: pluginsdk.ActionErrorConflict,
		},
		{
			name: "provider resolution unavailable", runtime: &Runtime{workflows: unavailableWorkflows},
			request:    &pluginsdk.PluginActionRequest{ActionKey: "repositories.list", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}},
			wantStatus: http.StatusServiceUnavailable, wantCode: pluginsdk.ActionErrorUnavailable,
		},
		{
			name: "verified task unavailable", runtime: &Runtime{workflows: missingTaskWorkflows},
			request:    &pluginsdk.PluginActionRequest{ActionKey: "pullrequests.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1", TaskID: "missing-task"}, Body: []byte(`{"title":"PR"}`)},
			wantStatus: http.StatusForbidden, wantCode: pluginsdk.ActionErrorPermissionDenied,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, actionErr := test.runtime.HandleAction(context.Background(), test.request)
			require.NoError(t, actionErr)
			require.Equal(t, test.wantStatus, response.Status)
			var body map[string]any
			require.NoError(t, json.Unmarshal(response.Body, &body))
			require.Equal(t, string(test.wantCode), body["code"])
		})
	}
}

func TestRuntime_MapsConnectionAndWatchValidationToTypedResponses(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	runtime := &Runtime{workflows: workflows}

	response, err := runtime.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "connection.save", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"product":"cloud","auth_method":"api_token","auth_identity":"dev@example.test","token":"token"}`),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, response.Status)
	require.JSONEq(t, `{"code":"invalid_argument","error":"Invalid Bitbucket action request."}`, string(response.Body))

	response, err = runtime.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "oauth.start", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, response.Status)
	require.JSONEq(t, `{"code":"conflict","error":"Bitbucket resource changed. Refresh and retry."}`, string(response.Body))

	response, err = runtime.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "watches.create", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}, Body: []byte(`{}`),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, response.Status)
	require.JSONEq(t, `{"code":"invalid_argument","error":"Invalid Bitbucket action request."}`, string(response.Body))
}

func TestRuntime_PreservesProviderRateLimitWithoutLeakingDetails(t *testing.T) {
	provider := &workflowProvider{
		pullRequest: testPullRequest(),
		reviewErr: &domain.ProviderHTTPError{
			Status: http.StatusTooManyRequests, RetryAfter: "30",
			Err: errors.New("secret upstream body"),
		},
	}
	workflows, err := NewWorkflows(newConnectionHost(), staticResolver{provider: provider})
	require.NoError(t, err)
	runtime := &Runtime{workflows: workflows}

	response, err := runtime.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "pullrequests.inspect",
		Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body:      []byte(`{"review_key":"workspace/repo#42"}`),
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, response.Status)
	require.Equal(t, "30", response.Headers["Retry-After"])
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body, &body))
	require.Equal(t, string(pluginsdk.ActionErrorRateLimited), body["code"])
	require.NotContains(t, string(response.Body), "secret upstream body")
}

func TestActionFailureClassificationUsesTypedWatchErrors(t *testing.T) {
	status, code, _, _ := classifyActionFailure(categorizeWatchActionError(watches.ErrWatchNotFound))
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "not_found", code)

	status, code, _, _ = classifyActionFailure(errors.New("watch not found"))
	require.Equal(t, http.StatusBadGateway, status, "message text must not define transport semantics")
	require.Equal(t, string(pluginsdk.ActionErrorUpstream), code)
}

func TestConnectionHealthUsesTypedProviderAuthenticationStatus(t *testing.T) {
	settings := ConnectionSettings{Product: domain.ProductCloud}
	auth := connectionResponse(settings, false, &domain.ProviderHTTPError{Status: http.StatusUnauthorized})
	unavailable := connectionResponse(settings, false, &domain.ProviderHTTPError{Status: http.StatusServiceUnavailable, Err: errors.New("authorization service")})

	require.Equal(t, "auth_required", auth["state"])
	require.Equal(t, "unavailable", unavailable["state"], "error wording must not masquerade as authentication failure")
}
