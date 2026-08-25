package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuthorizationCodeExchangeUsesPKCEAndRejectsCallbackReplay(t *testing.T) {
	states, err := NewStateManager([]byte("01234567890123456789012345678901"), NewMemoryPendingStore(), time.Now)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		require.Equal(t, "code-from-provider", r.Form.Get("code"))
		require.Equal(t, "https://kandev.example.test/callback", r.Form.Get("redirect_uri"))
		require.NotEmpty(t, r.Form.Get("code_verifier"))
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":3600}`))
	}))
	defer server.Close()
	tokenURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	authorizationURL, err := url.Parse("https://bitbucket.org/site/oauth2/authorize")
	require.NoError(t, err)
	redirectURL, err := url.Parse("https://kandev.example.test/callback")
	require.NoError(t, err)
	registration := OAuthRegistration{
		ClientID:         "client-id",
		ClientSecret:     "client-secret",
		AuthorizationURL: authorizationURL,
		TokenURL:         tokenURL,
		RedirectURL:      redirectURL,
	}

	request, err := StartAuthorization(context.Background(), states, CredentialScope{WorkspaceID: "workspace-a", Generation: 4}, registration, []string{"repository", "pullrequest"})
	require.NoError(t, err)
	require.Equal(t, "client-id", request.URL.Query().Get("client_id"))
	require.Equal(t, "code", request.URL.Query().Get("response_type"))
	require.Equal(t, "S256", request.URL.Query().Get("code_challenge_method"))
	require.NotEmpty(t, request.URL.Query().Get("state"))

	refresher := NewRefresher(server.Client(), time.Now)
	credential, err := refresher.ExchangeAuthorizationCode(context.Background(), states, request.State, "code-from-provider", registration)
	require.NoError(t, err)
	require.Equal(t, "access", credential.AccessToken)
	require.Equal(t, "refresh", credential.RefreshToken)

	_, err = refresher.ExchangeAuthorizationCode(context.Background(), states, request.State, "code-from-provider", registration)
	require.ErrorIs(t, err, ErrStateReplay)
}

func TestStartAuthorizationAllowsHTTPOnlyForLoopbackRedirects(t *testing.T) {
	authorizationURL, err := url.Parse("https://bitbucket.org/site/oauth2/authorize")
	require.NoError(t, err)

	tests := []struct {
		name        string
		redirectURL string
		wantError   bool
	}{
		{name: "https", redirectURL: "https://kandev.example.test/callback"},
		{name: "localhost", redirectURL: "http://localhost:38429/callback"},
		{name: "IPv4 loopback", redirectURL: "http://127.0.0.1:38429/callback"},
		{name: "IPv6 loopback", redirectURL: "http://[::1]:38429/callback"},
		{name: "non-loopback HTTP", redirectURL: "http://kandev.example.test/callback", wantError: true},
		{name: "credentials", redirectURL: "http://user@localhost:38429/callback", wantError: true},
		{name: "query", redirectURL: "http://localhost:38429/callback?next=other", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			redirectURL, parseErr := url.Parse(test.redirectURL)
			require.NoError(t, parseErr)
			states, stateErr := NewStateManager([]byte("01234567890123456789012345678901"), NewMemoryPendingStore(), time.Now)
			require.NoError(t, stateErr)
			_, startErr := StartAuthorization(context.Background(), states, CredentialScope{WorkspaceID: "workspace-a", Generation: 1}, OAuthRegistration{
				ClientID: "client-id", AuthorizationURL: authorizationURL, RedirectURL: redirectURL,
			}, nil)
			if test.wantError {
				require.Error(t, startErr)
				return
			}
			require.NoError(t, startErr)
		})
	}
}

func TestStartAuthorizationStillRequiresHTTPSAuthorizationEndpoint(t *testing.T) {
	authorizationURL, err := url.Parse("http://localhost:38429/authorize")
	require.NoError(t, err)
	redirectURL, err := url.Parse("http://localhost:38429/callback")
	require.NoError(t, err)
	states, err := NewStateManager([]byte("01234567890123456789012345678901"), NewMemoryPendingStore(), time.Now)
	require.NoError(t, err)

	_, err = StartAuthorization(context.Background(), states, CredentialScope{WorkspaceID: "workspace-a", Generation: 1}, OAuthRegistration{
		ClientID: "client-id", AuthorizationURL: authorizationURL, RedirectURL: redirectURL,
	}, nil)
	require.ErrorContains(t, err, "authorization endpoint must use HTTPS")
}
