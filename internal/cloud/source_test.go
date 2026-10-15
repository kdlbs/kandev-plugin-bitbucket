package cloud

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"kandev-plugin-bitbucket/internal/domain"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSourceTransportRejectsGitIdentityBeforeNetwork(t *testing.T) {
	calls := 0
	client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), Authentication: Authentication{Mode: AuthenticationAPIToken, Email: "x-bitbucket-api-token-auth"}, HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})}})
	gitCredential, gitErr := client.ResolveGitCredential(context.Background())
	require.NoError(t, gitErr)
	require.Equal(t, "x-bitbucket-api-token-auth", gitCredential.Username)
	require.Equal(t, "synthetic-token", gitCredential.Secret)
	_, err := client.SourceOperation(context.Background(), domain.Repository{ID: "repo", ProviderScope: "https://bitbucket.org", Namespace: "team", Slug: "repo"}, "feature/owned", "main", SourceInput{Operation: "pr_lookup"})
	require.Error(t, err)
	require.Zero(t, calls)
}

type sourceRoundTrip func(*http.Request) (*http.Response, error)

func (f sourceRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSourceTransportExactSourcePipelineLogAndRedaction(t *testing.T) {
	calls := 0
	client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), Authentication: Authentication{Mode: AuthenticationAPIToken, Email: "owner@example.test"}, HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		username, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "owner@example.test", username)
		require.Equal(t, "synthetic-token", password)
		payload := `{}`
		switch {
		case strings.Contains(r.URL.Path, "/refs/branches/"):
			payload = `{"name":"feature/owned","target":{"hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`
		case strings.HasSuffix(r.URL.Path, "/log"):
			payload = "synthetic-token " + strings.Repeat("L", sourceLogLimit)
		case strings.HasSuffix(r.URL.Path, "/steps/{step}"):
			payload = `{"uuid":"{step}","name":"test"}`
		default:
			payload = `{"uuid":"{pipeline}","target":{"type":"pipeline_ref_target","ref_type":"branch","ref_name":"feature/owned","commit":{"hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}})
	result, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pipeline_log", PipelineID: "{pipeline}", StepID: "{step}"})
	require.NoError(t, err)
	require.Equal(t, true, result["truncated"])
	require.NotContains(t, result["log"], "synthetic-token")
	require.Equal(t, 4, calls)
}

func sourceTestRepository() domain.Repository {
	return domain.Repository{ID: "repo", ProviderScope: "https://bitbucket.org", Namespace: "team", Slug: "repo"}
}

func TestSourceTransportPRDedupeIsExactAndBounded(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		wantError     bool
	}{
		{"exact", `{"values":[{"id":42,"state":"OPEN","source":{"branch":{"name":"feature/owned"},"repository":{"uuid":"repo"}},"destination":{"branch":{"name":"main"}}}]}`, false},
		{"more pages", `{"values":[],"next":"https://evil.test/secret"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Contains(t, r.URL.Query().Get("q"), `source.branch.name = "feature/owned"`)
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.payload))}, nil
			})}})
			result, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_create", Title: "title"})
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, true, result["reused"])
				require.Equal(t, 42, result["number"])
			}
		})
	}
}

func TestSourceTransportErrorsRedirectsExpiryAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"identity 10402", 401, "synthetic-token API token must be used with an atlassian registered email", "bitbucket_api_identity_unauthorized"},
		{"REST routing 10405", 404, "GraphQL failure synthetic-token", "bitbucket_rest_resource_unavailable"},
		{"redirect", 302, "synthetic-token", "bitbucket_rest_status_302"},
		{"oversized", 200, strings.Repeat("X", (1<<20)+1), "bitbucket_source_response_limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{"http://169.254.169.254/metadata"}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}})
			_, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_lookup"})
			require.ErrorContains(t, err, tc.want)
			require.NotContains(t, err.Error(), "synthetic-token")
			require.Equal(t, 1, calls)
		})
	}
}

func TestSourceTransportRejectsUnsafeInputsBeforeHTTP(t *testing.T) {
	for _, tc := range []struct {
		name  string
		repo  domain.Repository
		input SourceInput
	}{
		{"arbitrary pipeline URL", sourceTestRepository(), SourceInput{Operation: "pipeline_get", PipelineID: "https://evil.test/path"}},
		{"traversal", domain.Repository{ID: "repo", ProviderScope: "https://bitbucket.org", Namespace: "..", Slug: "repo"}, SourceInput{Operation: "pr_lookup"}},
		{"foreign origin", domain.Repository{ID: "repo", ProviderScope: "https://evil.test", Namespace: "team", Slug: "repo"}, SourceInput{Operation: "pr_lookup"}},
		{"write capability", sourceTestRepository(), SourceInput{Operation: "deploy"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})}})
			_, err := client.SourceOperation(context.Background(), tc.repo, "feature/owned", "main", tc.input)
			require.Error(t, err)
			require.Zero(t, calls)
		})
	}
}

func TestSourceTransportCreatesOnceThenReusesExactPR(t *testing.T) {
	created := false
	writes := 0
	pr := `{"id":42,"state":"OPEN","source":{"branch":{"name":"feature/owned"},"repository":{"uuid":"repo"}},"destination":{"branch":{"name":"main"}}}`
	client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
		payload := `{"values":[]}`
		if r.Method == http.MethodPost {
			writes++
			created = true
			payload = pr
			data, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(data), `"close_source_branch":false`)
			require.NotContains(t, string(data), "synthetic-token")
		} else if created {
			payload = `{"values":[` + pr + `]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}})
	for i := 0; i < 2; i++ {
		result, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_create", Title: "A change"})
		require.NoError(t, err)
		require.Equal(t, i == 1, result["reused"])
	}
	require.Equal(t, 1, writes)
}

func TestSourceTransportRejectsExpiredCredential(t *testing.T) {
	calls := 0
	client := NewClient(ClientOptions{TokenSource: expiringTokenSource{token: "synthetic-token", expiresAt: time.Now().Add(-time.Minute)}, HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("unexpected") })}})
	_, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_lookup"})
	require.ErrorContains(t, err, "credential_unavailable")
	require.Zero(t, calls)
}

func TestSourceTransportRejectsDifferentSourcePipelineAndPR(t *testing.T) {
	for _, operation := range []string{"pr_get", "pipeline_log"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				payload := `{"uuid":"{pipeline}","target":{"type":"pipeline_ref_target","ref_type":"branch","ref_name":"other","commit":{"hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}`
				if strings.Contains(r.URL.Path, "/branches/") {
					payload = `{"name":"feature/owned","target":{"hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`
				}
				if strings.Contains(r.URL.Path, "/pullrequests/") {
					payload = `{"id":42,"source":{"branch":{"name":"other"},"repository":{"uuid":"repo"}},"destination":{"branch":{"name":"main"}}}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
			})}})
			_, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: operation, Number: 42, PipelineID: "{pipeline}", StepID: "{step}"})
			require.ErrorContains(t, err, "source_mismatch")
			require.LessOrEqual(t, calls, 2)
		})
	}
}

func TestSourceTransportReadOperationsNeverFollowPagination(t *testing.T) {
	const head = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, operation := range []string{"pr_get", "pipelines", "pipeline_get", "pipeline_steps", "statuses"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "api.bitbucket.org", r.URL.Host)
				pipeline := `{"uuid":"{pipeline}","target":{"type":"pipeline_ref_target","ref_type":"branch","ref_name":"feature/owned","commit":{"hash":"` + head + `"}}}`
				payload := pipeline
				switch {
				case strings.Contains(r.URL.Path, "/branches/"):
					payload = `{"name":"feature/owned","target":{"hash":"` + head + `"}}`
				case strings.HasSuffix(r.URL.Path, "/pullrequests/42"):
					payload = `{"id":42,"source":{"branch":{"name":"feature/owned"},"repository":{"uuid":"repo"}},"destination":{"branch":{"name":"main"}}}`
				case strings.HasSuffix(r.URL.Path, "/pipelines"):
					payload = `{"values":[` + pipeline + `],"next":"https://evil.test/page"}`
				case strings.HasSuffix(r.URL.Path, "/steps"):
					payload = `{"values":[{"uuid":"{step}","name":"test"}],"next":"https://evil.test/page"}`
				case strings.HasSuffix(r.URL.Path, "/statuses"):
					payload = `{"values":[{"key":"tests","name":"test","state":"SUCCESSFUL","description":"ok"}],"next":"https://evil.test/page"}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
			})}})
			result, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: operation, Number: 42, PipelineID: "{pipeline}"})
			require.NoError(t, err)
			require.LessOrEqual(t, calls, 3)
			if operation == "pipelines" || operation == "pipeline_steps" || operation == "statuses" {
				require.Equal(t, true, result["truncated"])
			}
		})
	}
}

func TestSourceTransportGETHasNoRequestPayload(t *testing.T) {
	client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
		require.Zero(t, r.ContentLength)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"values":[]}`))}, nil
	})}})
	_, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_lookup"})
	require.NoError(t, err)
}

func TestSourceTransportRejectsMismatchedPullRequestNumber(t *testing.T) {
	client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":43,"source":{"branch":{"name":"feature/owned"},"repository":{"uuid":"repo"}},"destination":{"branch":{"name":"main"}}}`))}, nil
	})}})
	_, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_get", Number: 42})
	require.ErrorContains(t, err, "source_mismatch")
}

func TestSourceTransportDoesNotCreateWhenLookupIdentityIsIncomplete(t *testing.T) {
	for _, payload := range []string{`{"values":[{"id":42,"state":"OPEN","source":{"branch":{"name":"feature/owned"}},"destination":{"branch":{"name":"main"}}}]}`, `{}`, `{"values":null}`} {
		t.Run(payload, func(t *testing.T) {
			writes := 0
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPost {
					writes++
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
			})}})
			_, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: "pr_create", Title: "change"})
			require.ErrorContains(t, err, "lookup_incomplete")
			require.Zero(t, writes)
		})
	}
}

func TestSourceTransportPipelineTargetKinds(t *testing.T) {
	const head = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, tc := range []struct {
		name, operation, target string
		allowed                 bool
	}{
		{"PR get", "pipeline_get", `{"type":"pipeline_pullrequest_target","source":"feature/owned","commit":{"hash":"` + head + `"}}`, true},
		{"PR list", "pipelines", `{"type":"pipeline_pullrequest_target","source":"feature/owned","commit":{"hash":"` + head + `"}}`, true},
		{"tag", "pipeline_get", `{"type":"pipeline_ref_target","ref_type":"tag","ref_name":"feature/owned","commit":{"hash":"` + head + `"}}`, false},
		{"unknown", "pipeline_get", `{"type":"unknown","ref_name":"feature/owned","commit":{"hash":"` + head + `"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(ClientOptions{TokenSource: staticTokenSource("synthetic-token"), HTTPClient: &http.Client{Transport: sourceRoundTrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, r.Method)
				payload := `{"uuid":"{pipeline}","target":` + tc.target + `}`
				if strings.Contains(r.URL.Path, "/branches/") {
					payload = `{"name":"feature/owned","target":{"hash":"` + head + `"}}`
				} else if tc.operation == "pipelines" {
					require.Empty(t, r.URL.Query().Get("target.ref_name"))
					payload = `{"values":[` + payload + `]}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
			})}})
			result, err := client.SourceOperation(context.Background(), sourceTestRepository(), "feature/owned", "main", SourceInput{Operation: tc.operation, PipelineID: "{pipeline}"})
			if !tc.allowed {
				require.ErrorContains(t, err, "source_mismatch")
				return
			}
			require.NoError(t, err)
			if tc.operation == "pipelines" {
				require.Len(t, result["pipelines"], 1)
			} else {
				require.Equal(t, "feature/owned", result["source_branch"])
			}
		})
	}
}
