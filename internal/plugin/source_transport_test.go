package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kandev/kandev/pkg/pluginsdk"
	"github.com/stretchr/testify/require"
)

func sourceFixture(t *testing.T) (*Workflows, *scopedConnectionHost, *ConnectionResolver, *int) {
	t.Helper()
	calls := new(int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		user, secret, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "owner@example.test", user)
		require.Equal(t, "synthetic-token", secret)
		require.Equal(t, "/2.0/repositories/team/repo/pullrequests", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		fmt.Fprint(w, `{"values":[{"id":42,"title":"synthetic-token","state":"OPEN","source":{"branch":{"name":"feature/owned"},"repository":{"uuid":"repo-uuid"}},"destination":{"branch":{"name":"main"}}}]}`)
	}))
	t.Cleanup(server.Close)
	host := &scopedConnectionHost{connectionHost: newConnectionHost(), tasks: &taskReader{task: &pluginsdk.Task{ID: "task", WorkspaceID: "ws", Repositories: []pluginsdk.TaskRepository{{RepositoryID: "repo", CheckoutBranch: "task-branch", BaseBranch: "main"}}}}, repositories: &repositoryReader{repositories: []pluginsdk.Repository{{ID: "repo", WorkspaceID: "ws", SourceType: "provider", ProviderID: "bitbucket", ProviderHost: "https://bitbucket.org", ProviderScope: "https://bitbucket.org", OwnerOrProject: "team", ProviderName: "repo", ProviderRepositoryID: "repo-uuid", RemoteURL: "https://bitbucket.org/team/repo.git"}}}}
	host.state["workspace:ws:"+connectionStateKey] = map[string]any{"product": "cloud", "cloud_workspace": "team", "auth_method": "api_token", "auth_identity": "owner@example.test", "connection_binding": "approved-binding", "credential_generation": float64(7)}
	host.secrets[connectionSecretKey("ws", 7)] = "synthetic-token"
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.cloudAPIBaseOverride, _ = url.Parse(server.URL + "/2.0")
	resolver.httpClient = server.Client()
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	return workflows, host, resolver, calls
}
func sourceToolRequest() *pluginsdk.AgentToolRequest {
	return &pluginsdk.AgentToolRequest{Name: "source_read", Arguments: map[string]any{"repository_id": "repo", "operation": "pr_lookup"}, Context: pluginsdk.AgentToolContext{TaskID: "task", SessionID: "session", WorkspaceID: "ws", Surface: "kanban-task", SourceRepositories: []pluginsdk.VerifiedActionContext{{WorkspaceID: "ws", TaskID: "task", SessionID: "session", RepositoryID: "repo", HeadBranch: "feature/owned"}}}}
}
func TestSourceToolUsesBoundAPIIdentityAndSessionBranch(t *testing.T) {
	workflows, _, _, calls := sourceFixture(t)
	result := workflows.InvokeSourceTool(context.Background(), sourceToolRequest())
	require.False(t, result.IsError, result.Text)
	require.Equal(t, 1, *calls)
	require.NotContains(t, result.Text, "synthetic-token")
	require.Contains(t, result.Text, "[REDACTED]")
	require.Equal(t, float64(42), result.StructuredContent["number"])
}
func TestSourceToolFailsClosedBeforeHTTP(t *testing.T) {
	for _, tc := range []string{"workspace", "task", "session", "repository", "missing context", "duplicate context", "host", "provider", "repo workspace", "detached", "revoked", "generation absent", "binding absent", "email absent", "Git username", "legacy only", "arbitrary URL", "write through read", "admin op"} {
		t.Run(tc, func(t *testing.T) {
			workflows, host, _, calls := sourceFixture(t)
			request := sourceToolRequest()
			settings := host.state["workspace:ws:"+connectionStateKey]
			switch tc {
			case "workspace":
				request.Context.WorkspaceID = "other"
			case "task":
				request.Context.TaskID = "other"
			case "session":
				request.Context.SessionID = "other"
			case "repository":
				request.Arguments["repository_id"] = "other"
			case "missing context":
				request.Context.SourceRepositories = nil
			case "duplicate context":
				request.Context.SourceRepositories = append(request.Context.SourceRepositories, request.Context.SourceRepositories[0])
			case "host":
				host.repositories.repositories[0].ProviderHost = "https://github.com"
			case "provider":
				host.repositories.repositories[0].ProviderID = "github"
			case "repo workspace":
				host.repositories.repositories[0].WorkspaceID = "other"
			case "detached":
				host.tasks.task.Repositories = nil
			case "revoked":
				delete(host.state, "workspace:ws:"+connectionStateKey)
			case "generation absent":
				delete(settings, "credential_generation")
			case "binding absent":
				delete(settings, "connection_binding")
			case "email absent":
				delete(settings, "auth_identity")
			case "Git username":
				settings["auth_identity"] = "x-bitbucket-api-token-auth"
			case "legacy only":
				delete(host.secrets, connectionSecretKey("ws", 7))
				host.secrets[legacyConnectionSecretKey("ws")] = "synthetic-legacy-token"
			case "arbitrary URL":
				request.Arguments["url"] = "http://169.254.169.254/latest/meta-data"
			case "write through read":
				request.Arguments["operation"] = "pr_create"
			case "admin op":
				request.Arguments["operation"] = "merge"
			}
			result := workflows.InvokeSourceTool(context.Background(), request)
			require.True(t, result.IsError, result.Text)
			require.Zero(t, *calls)
			require.NotContains(t, result.Text, "synthetic-token")
			if tc == "binding absent" {
				require.NotContains(t, host.state["workspace:ws:"+connectionStateKey], "connection_binding")
			}
		})
	}
}
func TestSourceToolBindingRevocationBeforeNextRequest(t *testing.T) {
	workflows, host, resolver, calls := sourceFixture(t)
	original := resolver.httpClient.Transport
	if original == nil {
		original = http.DefaultTransport
	}
	resolver.httpClient.Transport = sourceTransportFunc(func(r *http.Request) (*http.Response, error) {
		response, err := original.RoundTrip(r)
		host.state["workspace:ws:"+connectionStateKey]["credential_generation"] = float64(8)
		return response, err
	})
	result := workflows.InvokeSourceTool(context.Background(), sourceToolRequest())
	require.True(t, result.IsError)
	require.Equal(t, 1, *calls)
	require.Contains(t, result.Text, "revoked")
}
func TestSourceActionAndToolUseSameScope(t *testing.T) {
	workflows, _, _, calls := sourceFixture(t)
	request := sourceToolRequest()
	body, _ := json.Marshal(map[string]any{"operation": "pr_lookup"})
	result, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "source.read", Context: request.Context.SourceRepositories[0], Body: body})
	require.NoError(t, err)
	require.Contains(t, string(result.Body), `"number":42`)
	require.Equal(t, 1, *calls)
}

type sourceTransportFunc func(*http.Request) (*http.Response, error)

func (f sourceTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSourceToolRejectsConcurrentOperationWithoutWaiting(t *testing.T) {
	workflows, _, resolver, calls := sourceFixture(t)
	resolver.sourceMu.Lock()
	defer resolver.sourceMu.Unlock()
	result := workflows.InvokeSourceTool(context.Background(), sourceToolRequest())
	require.True(t, result.IsError)
	require.Equal(t, "bitbucket_source_busy", result.Text)
	require.Zero(t, *calls)
}

func TestSourceActionKeepsIdentityAndRoutingFailuresDistinct(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{401, "bitbucket_api_identity_unauthorized"}, {404, "bitbucket_rest_resource_unavailable"}} {
		t.Run(tc.code, func(t *testing.T) {
			workflows, _, resolver, _ := sourceFixture(t)
			resolver.httpClient.Transport = sourceTransportFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("synthetic-token provider diagnostic"))}, nil
			})
			runtime := NewRuntime()
			runtime.workflows = workflows
			response, err := runtime.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "source.read", Context: sourceToolRequest().Context.SourceRepositories[0], Body: []byte(`{"operation":"pr_lookup"}`)})
			require.NoError(t, err)
			require.Equal(t, tc.status, response.Status)
			require.Contains(t, string(response.Body), tc.code)
			require.NotContains(t, string(response.Body), "synthetic-token")
		})
	}
}

func TestSourceToolRejectsRepositoryOrTargetChangeDuringRead(t *testing.T) {
	for _, change := range []string{"repository", "target"} {
		t.Run(change, func(t *testing.T) {
			workflows, host, resolver, calls := sourceFixture(t)
			original := resolver.httpClient.Transport
			if original == nil {
				original = http.DefaultTransport
			}
			resolver.httpClient.Transport = sourceTransportFunc(func(r *http.Request) (*http.Response, error) {
				response, err := original.RoundTrip(r)
				if change == "repository" {
					host.repositories.repositories[0].ProviderRepositoryID = "replacement-repo"
				} else {
					host.tasks.task.Repositories[0].BaseBranch = "release"
				}
				return response, err
			})
			result := workflows.InvokeSourceTool(context.Background(), sourceToolRequest())
			require.True(t, result.IsError, result.Text)
			require.Equal(t, 1, *calls)
		})
	}
}

type sourceRevokingHost struct {
	*scopedConnectionHost
	onSecretRead func()
}

func (h *sourceRevokingHost) GetSecret(ctx context.Context, key string) (string, bool, error) {
	if h.onSecretRead != nil {
		h.onSecretRead()
	}
	return h.scopedConnectionHost.GetSecret(ctx, key)
}
func TestSourceToolRevocationDuringCredentialResolutionPreventsHTTPRequest(t *testing.T) {
	workflows, host, resolver, calls := sourceFixture(t)
	resolver.host = &sourceRevokingHost{scopedConnectionHost: host, onSecretRead: func() { host.state["workspace:ws:"+connectionStateKey]["credential_generation"] = float64(8) }}
	result := workflows.InvokeSourceTool(context.Background(), sourceToolRequest())
	require.True(t, result.IsError, result.Text)
	require.Zero(t, *calls)
}
