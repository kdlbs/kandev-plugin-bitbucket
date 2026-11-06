package cloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"kandev-plugin-bitbucket/internal/domain"
)

func TestListRepositoriesFollowsCloudCursorAndRespectsLimit(t *testing.T) {
	firstPage, err := os.ReadFile("testdata/repositories-first.json")
	require.NoError(t, err)
	secondPage, err := os.ReadFile("testdata/repositories-second.json")
	require.NoError(t, err)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/2.0/repositories/acme", r.URL.Path)
		require.Equal(t, "Bearer cloud-token", r.Header.Get("Authorization"))
		switch r.URL.Query().Get("after") {
		case "":
			require.Equal(t, "2", r.URL.Query().Get("pagelen"))
			_, _ = w.Write(firstPage)
		case "cursor-2":
			_, _ = w.Write(secondPage)
		default:
			t.Fatalf("unexpected cursor %q", r.URL.Query().Get("after"))
		}
		requests++
	}))
	defer server.Close()

	apiBase, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: apiBase, TokenSource: staticTokenSource("cloud-token")})

	repositories, err := client.ListRepositories(context.Background(), "acme", 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(repositories))
	require.Equal(t, "widgets", repositories[0].Slug)
	require.Equal(t, "https://bitbucket.org/acme/widgets.git", repositories[0].CloneURL.String())
	require.Equal(t, "main", repositories[0].DefaultBranch)
	require.Equal(t, "dashboard", repositories[1].Slug)
	require.Equal(t, 2, requests)
}

func TestSearchRepositoriesSendsCloudServerSideFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, `/2.0/repositories/acme`, r.URL.Path)
		require.Equal(t, `name ~ "widget\" api"`, r.URL.Query().Get("q"))
		require.Equal(t, "3", r.URL.Query().Get("pagelen"))
		_, _ = w.Write([]byte(`{"values":[]}`))
	}))
	defer server.Close()

	apiBase, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: apiBase, TokenSource: staticTokenSource("cloud-token")})

	_, err = client.SearchRepositories(context.Background(), "acme", `widget" api`, 3)
	require.NoError(t, err)
}

func TestRepositoryCursorIsBoundToCloudWorkspaceAndQuery(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") == "" {
			_, _ = w.Write([]byte(`{"values":[],"next":"` + server.URL + `/2.0/repositories/acme?after=next&q=name+~+%22widgets%22"}`))
			return
		}
		_, _ = w.Write([]byte(`{"values":[]}`))
	}))
	defer server.Close()

	apiBase, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: apiBase, TokenSource: staticTokenSource("cloud-token")})
	page, err := client.ListRepositoriesPage(context.Background(), "acme", domain.RepositoryQuery{Text: "widgets", Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, page.NextCursor)

	_, err = client.ListRepositoriesPage(context.Background(), "other", domain.RepositoryQuery{Text: "widgets", Limit: 10, Cursor: page.NextCursor})
	require.ErrorContains(t, err, "invalid Bitbucket Cloud repository cursor")
	_, err = client.ListRepositoriesPage(context.Background(), "acme", domain.RepositoryQuery{Text: "dashboard", Limit: 10, Cursor: page.NextCursor})
	require.ErrorContains(t, err, "invalid Bitbucket Cloud repository cursor")
	_, err = client.ListRepositoriesPage(context.Background(), "acme", domain.RepositoryQuery{Text: "widgets", Limit: 25, Cursor: page.NextCursor})
	require.ErrorContains(t, err, "invalid Bitbucket Cloud repository cursor")
}

func TestListRepositoriesRetriesRateLimitsWithBoundedBackoff(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"values":[]}`))
	}))
	defer server.Close()
	apiBase, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{
		APIBase:     apiBase,
		TokenSource: staticTokenSource("cloud-token"),
		RetryDelay:  func(int, time.Duration) time.Duration { return 0 },
	})

	repositories, err := client.ListRepositories(context.Background(), "acme", 1)
	require.NoError(t, err)
	require.Empty(t, repositories)
	require.Equal(t, int32(2), attempts.Load())
}

func TestCloudAuthenticationSelectsExactRESTAndGitSchemes(t *testing.T) {
	tests := []struct {
		name              string
		authentication    Authentication
		wantAuthorization string
		wantGitUsername   string
	}{
		{
			name:              "api token",
			authentication:    Authentication{Mode: AuthenticationAPIToken, Email: "dev@example.test"},
			wantAuthorization: "Basic ZGV2QGV4YW1wbGUudGVzdDphcGktdG9rZW4=",
			wantGitUsername:   "x-bitbucket-api-token-auth",
		},
		{
			name:              "access token",
			authentication:    Authentication{Mode: AuthenticationAccessToken},
			wantAuthorization: "Bearer oauth-token",
			wantGitUsername:   "x-token-auth",
		},
		{
			name:              "oauth",
			authentication:    Authentication{Mode: AuthenticationOAuth},
			wantAuthorization: "Bearer oauth-token",
			wantGitUsername:   "x-token-auth",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, test.wantAuthorization, r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"values":[]}`))
			}))
			defer server.Close()
			apiBase, err := url.Parse(server.URL + "/2.0")
			require.NoError(t, err)
			token := "oauth-token"
			if test.authentication.Mode == AuthenticationAPIToken {
				token = "api-token"
			}
			client := NewClient(ClientOptions{APIBase: apiBase, HTTPClient: server.Client(), TokenSource: staticTokenSource(token), Authentication: test.authentication})

			_, err = client.ListRepositories(context.Background(), "acme", 1)
			require.NoError(t, err)
			credential, err := client.ResolveGitCredential(context.Background())
			require.NoError(t, err)
			require.Equal(t, test.wantGitUsername, credential.Username)
			require.Equal(t, token, credential.Secret)
			require.False(t, credential.ExpiresAt.IsZero())
		})
	}
}

func TestCloudGitCredentialKeepsOAuthTokenExpiryInMemory(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC().Round(0)
	client := NewClient(ClientOptions{
		TokenSource:    expiringTokenSource{token: "oauth-token", expiresAt: expiresAt},
		Authentication: Authentication{Mode: AuthenticationOAuth},
	})

	credential, err := client.ResolveGitCredential(context.Background())
	require.NoError(t, err)
	require.Equal(t, expiresAt, credential.ExpiresAt)
}

func TestCloudRejectsCredentialedOrOutOfContextNextPages(t *testing.T) {
	apiBase, err := url.Parse("https://api.bitbucket.org/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: apiBase})
	for _, rawNext := range []string{
		"https://token@api.bitbucket.org/2.0/repositories/acme?page=2",
		"https://api.bitbucket.org/other/repositories/acme?page=2",
	} {
		_, err := client.nextURL(apiBase, rawNext)
		require.Error(t, err)
	}
}

type staticTokenSource string

func (s staticTokenSource) AccessToken(context.Context) (string, error) {
	return string(s), nil
}

type expiringTokenSource struct {
	token     string
	expiresAt time.Time
}

func (s expiringTokenSource) AccessToken(context.Context) (string, error) {
	return s.token, nil
}

func (s expiringTokenSource) AccessTokenCredential(context.Context) (string, time.Time, error) {
	return s.token, s.expiresAt, nil
}
