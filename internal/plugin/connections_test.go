package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"kandev-plugin-bitbucket/internal/auth"
	"kandev-plugin-bitbucket/internal/datacenter"
	"kandev-plugin-bitbucket/internal/domain"

	"github.com/kandev/kandev/pkg/pluginsdk"
	"github.com/stretchr/testify/require"
)

func TestConnectionResolver_PersistsOnlyCredentialFreeSettings(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test/bitbucket", AuthMethod: "user_pat", AuthIdentity: "dev", Token: "never-in-state",
	})
	require.NoError(t, err)
	require.Equal(t, domain.ProductDataCenter, settings.Product)
	require.Equal(t, "https://bitbucket.example.test/bitbucket", settings.BaseURL)
	require.NotContains(t, flatten(host.state), "never-in-state")
	require.Contains(t, host.secrets[connectionSecretKey("workspace-1", settings.CredentialGeneration)], "never-in-state")
}

func TestConnectionResolver_CredentialGenerationChangesOnSaveAndDisappearsOnDisconnect(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "first-token",
	})
	require.NoError(t, err)
	second, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "rotated-token",
	})
	require.NoError(t, err)
	require.NotZero(t, first.CredentialGeneration)
	require.Greater(t, second.CredentialGeneration, first.CredentialGeneration)
	require.NotEmpty(t, first.ConnectionBinding)
	require.Equal(t, first.ConnectionBinding, second.ConnectionBinding, "credential rotation on the same connection must keep watches bound")

	require.NoError(t, resolver.Disconnect(context.Background(), "workspace-1"))
	_, found, err := resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, err)
	require.False(t, found)
}

func TestConnectionResolver_ConnectionBindingChangesAcrossReplacementAndReconnect(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "first-token",
	})
	require.NoError(t, err)
	replacement, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "other", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "other-token",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.ConnectionBinding, replacement.ConnectionBinding)

	require.NoError(t, resolver.Disconnect(context.Background(), "workspace-1"))
	reconnected, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "new-token",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.ConnectionBinding, reconnected.ConnectionBinding, "a deleted connection epoch must never be reused")
}

func TestConnectionResolver_DisconnectRevokesCachedOAuthRefreshGeneration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer server.Close()
	tokenURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	scope := oauthCredentialScope("workspace-1", settings)
	registration := auth.OAuthRegistration{ClientID: "client-id", ClientSecret: "client-secret", TokenURL: tokenURL}
	_, err = resolver.oauthRefresher().Refresh(context.Background(), scope, registration, "old-refresh")
	require.NoError(t, err)

	require.NoError(t, resolver.Disconnect(context.Background(), "workspace-1"))
	_, err = resolver.oauthRefresher().Refresh(context.Background(), scope, registration, "old-refresh")
	require.ErrorIs(t, err, auth.ErrCredentialRevoked)
}

func TestConnectionResolver_TargetReplacementRevokesPreviousOAuthConnectionEpoch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer server.Close()
	tokenURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	registration := auth.OAuthRegistration{ClientID: "client-id", ClientSecret: "client-secret", TokenURL: tokenURL}
	firstScope := oauthCredentialScope("workspace-1", first)
	_, err = resolver.oauthRefresher().Refresh(context.Background(), firstScope, registration, "old-refresh")
	require.NoError(t, err)

	replacement, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "other", AuthMethod: "oauth",
	})
	require.NoError(t, err)
	require.Equal(t, first.OAuthGeneration, replacement.OAuthGeneration)
	require.NotEqual(t, first.ConnectionBinding, replacement.ConnectionBinding)
	_, err = resolver.oauthRefresher().Refresh(context.Background(), firstScope, registration, "old-refresh")
	require.ErrorIs(t, err, auth.ErrCredentialRevoked)
	_, err = resolver.oauthRefresher().Refresh(context.Background(), oauthCredentialScope("workspace-1", replacement), registration, "old-refresh")
	require.NoError(t, err)
}

func TestConnectionResolver_RejectsCredentialBearingBaseURL(t *testing.T) {
	resolver, err := NewConnectionResolver(newConnectionHost())
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://token@bitbucket.example.test", Token: "token",
	})
	require.Error(t, err)
	require.NotContains(t, strings.ToLower(err.Error()), "token")
}

func TestConnectionResolver_RejectsUnregisteredCredentialScopeOnConfiguredCloneHost(t *testing.T) {
	resolver, err := NewConnectionResolver(newConnectionHost())
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test/bitbucket", AuthMethod: "user_pat", AuthIdentity: "dev", Token: "token"})
	require.NoError(t, err)
	err = resolver.ValidateGitCredentialScope(context.Background(), GitCredentialScope{WorkspaceID: "workspace-1", Host: "bitbucket.example.test", Path: "/bitbucket/scm/ENG/repo.git"})
	require.ErrorIs(t, err, ErrCredentialUnavailable)
	err = resolver.ValidateGitCredentialScope(context.Background(), GitCredentialScope{WorkspaceID: "workspace-1", Host: "attacker.example.test", Path: "/bitbucket/scm/ENG/repo.git"})
	require.ErrorIs(t, err, ErrCredentialUnavailable)
}

func TestConnectionResolver_RejectsUnsupportedHTTPTokenMode(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "http_token", Token: "token",
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_RequiresInitialTokenCredential(t *testing.T) {
	for _, test := range []struct {
		name  string
		input ConnectionInput
	}{
		{name: "cloud api token", input: ConnectionInput{Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test"}},
		{name: "data center user PAT", input: ConnectionInput{Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "user_pat", AuthIdentity: "dev"}},
		{name: "data center project token", input: ConnectionInput{Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "project_token"}},
		{name: "data center repository token", input: ConnectionInput{Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "repository_token"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			host := newConnectionHost()
			resolver, err := NewConnectionResolver(host)
			require.NoError(t, err)
			_, err = resolver.Save(context.Background(), "workspace-1", test.input)
			require.Error(t, err)
			require.Empty(t, host.state)
			require.Empty(t, host.secrets)
		})
	}
}

func TestConnectionResolver_RejectsCloudAPIBaseOverride(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, BaseURL: "https://attacker.example.test/2.0", CloudWorkspace: "acme",
		AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "token",
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_UsesCloudAPITokenIdentityForAPIAndGit(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		email, token, ok := request.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "dev@example.test", email)
		require.Equal(t, "api-token", token)
		require.Equal(t, "/2.0/repositories/acme", request.URL.Path)
		require.Equal(t, `name ~ "widget"`, request.URL.Query().Get("q"))
		_, _ = writer.Write([]byte(`{"values":[]}`))
	}))
	defer server.Close()

	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	resolver.cloudAPIBaseOverride, err = url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme",
		AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "api-token",
	})
	require.NoError(t, err)
	require.Equal(t, "dev@example.test", settings.AuthIdentity)
	require.NotContains(t, flatten(host.state), "api-token")

	provider, err := resolver.Provider(context.Background(), "workspace-1")
	require.NoError(t, err)
	_, err = provider.ListRepositories(context.Background(), "widget", 1)
	require.NoError(t, err)
	credential, err := provider.ResolveGitCredential(context.Background())
	require.NoError(t, err)
	require.Equal(t, "x-bitbucket-api-token-auth", credential.Username)
	require.Equal(t, "api-token", credential.Secret)
}

func TestConnectionResolver_RejectsAPIAuthWithoutIdentityBeforeStoringToken(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", Token: "api-token",
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_RollsBackNewSecretsWhenConnectionStateWriteFails(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "old-token",
	})
	require.NoError(t, err)
	host.setStateErr = errors.New("state unavailable")

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.Error(t, err)
	settings, found, loadErr := resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, loadErr)
	require.True(t, found)
	require.Equal(t, "api_token", settings.AuthMethod)
	require.Equal(t, "old-token", host.secrets[connectionSecretKey("workspace-1", first.CredentialGeneration)])
	_, registrationSaved := host.secrets[oauthRegistrationSecretKey("workspace-1", 1)]
	require.False(t, registrationSaved)
}

func TestConnectionResolver_CompensatesStagedGenerationAfterCallerCancellation(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token",
		AuthIdentity: "dev@example.test", Token: "old-token",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	host.setStateErr = errors.New("state unavailable")
	host.beforeSetStateError = cancel
	_, err = resolver.Save(ctx, "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token",
		AuthIdentity: "dev@example.test", Token: "new-token",
	})
	require.Error(t, err)
	require.ErrorIs(t, ctx.Err(), context.Canceled)

	oldKey := connectionSecretKey("workspace-1", first.CredentialGeneration)
	stagedKey := connectionSecretKey("workspace-1", first.CredentialGeneration+1)
	require.Equal(t, "old-token", host.secrets[oldKey])
	_, stagedFound := host.secrets[stagedKey]
	require.False(t, stagedFound)
	require.Contains(t, host.deletedSecretKeys, stagedKey)
}

func TestConnectionResolver_CommitsNewGenerationAndRetriesOldSecretRevocation(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "old-token",
	})
	require.NoError(t, err)
	oldKey := connectionSecretKey("workspace-1", first.CredentialGeneration)
	host.deleteSecretErrKey = oldKey

	next, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	require.Equal(t, "oauth", next.AuthMethod)
	settings, found, loadErr := resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, loadErr)
	require.True(t, found)
	require.Equal(t, "oauth", settings.AuthMethod)
	require.Contains(t, settings.PendingSecretRevocations, oldKey)
	require.Equal(t, "old-token", host.secrets[oldKey])
	require.Contains(t, host.secrets[oauthRegistrationSecretKey("workspace-1", settings.OAuthGeneration)], "client-secret")

	host.deleteSecretErrKey = ""
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
	})
	require.NoError(t, err)
	_, oldSecretFound := host.secrets[oldKey]
	require.False(t, oldSecretFound)
	settings, found, loadErr = resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, loadErr)
	require.True(t, found)
	require.Empty(t, settings.PendingSecretRevocations)
}

func TestConnectionResolver_ReusesSavedOAuthRegistrationWithoutSecretReentry(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	first, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)

	second, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	require.Equal(t, first.OAuthGeneration, second.OAuthGeneration)
	require.Contains(t, host.secrets[oauthRegistrationSecretKey("workspace-1", second.OAuthGeneration)], "client-secret")
}

func TestConnectionResolver_AllowsHTTPOnlyForLoopbackOAuthRedirects(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		wantError   bool
	}{
		{name: "https", redirectURL: "https://kandev.example.test/callback"},
		{name: "localhost", redirectURL: "http://localhost:38429/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback"},
		{name: "IPv4 loopback", redirectURL: "http://127.0.0.1:38429/callback"},
		{name: "IPv6 loopback", redirectURL: "http://[::1]:38429/callback"},
		{name: "non-loopback HTTP", redirectURL: "http://kandev.example.test/callback", wantError: true},
		{name: "credentials", redirectURL: "http://user@localhost:38429/callback", wantError: true},
		{name: "query", redirectURL: "http://localhost:38429/callback?next=other", wantError: true},
		{name: "fragment", redirectURL: "http://localhost:38429/callback#fragment", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := newConnectionHost()
			resolver, err := NewConnectionResolver(host)
			require.NoError(t, err)
			_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
				Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
				OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: test.redirectURL,
			})
			if test.wantError {
				require.Error(t, err)
				require.Empty(t, host.state)
				require.Empty(t, host.secrets)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, host.state)
			require.NotEmpty(t, host.secrets)
		})
	}
}

func TestConnectionResolver_RejectsOAuthRegistrationReuseWhenSecretWasRevoked(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	delete(host.secrets, oauthRegistrationSecretKey("workspace-1", settings.OAuthGeneration))

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
	})
	require.Error(t, err)
}

func TestConnectionResolver_OwnsOneOAuthRefresher(t *testing.T) {
	resolver, err := NewConnectionResolver(newConnectionHost())
	require.NoError(t, err)
	require.Same(t, resolver.oauthRefresher(), resolver.oauthRefresher())
}

func TestConnectionResolver_ExchangesBYOOAuthCallbackOnceAndUsesRotatingCredential(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method + " " + request.URL.Path {
		case http.MethodPost + " /token":
			clientID, clientSecret, ok := request.BasicAuth()
			require.True(t, ok)
			require.Equal(t, "client-id", clientID)
			require.Equal(t, "client-secret", clientSecret)
			require.NotEmpty(t, request.FormValue("code_verifier"))
			_, _ = writer.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600}`))
		case http.MethodGet + " /2.0/user":
			require.Equal(t, "Bearer access-token", request.Header.Get("Authorization"))
			_, _ = writer.Write([]byte(`{}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	resolver.cloudAPIBaseOverride, err = url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	authorizationEndpoint, err := url.Parse(server.URL + "/authorize")
	require.NoError(t, err)
	tokenEndpoint, err := url.Parse(server.URL + "/token")
	require.NoError(t, err)
	resolver.oauthEndpointOverride = &oauthEndpointPair{authorization: authorizationEndpoint, token: tokenEndpoint}
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	responseState := connectionResponse(settings, false, nil)
	require.Equal(t, "client-id", responseState["oauth_client_id"])
	require.NotContains(t, flatten(responseState), "client-secret")

	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	started, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "oauth.start", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}})
	require.NoError(t, err)
	var startResponse struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(started.Body, &startResponse))
	authorizationURL, err := url.Parse(startResponse.URL)
	require.NoError(t, err)
	state := authorizationURL.Query().Get("state")
	require.NotEmpty(t, state)
	require.Equal(t, "S256", authorizationURL.Query().Get("code_challenge_method"))

	response, err := workflows.HandleWebhook(context.Background(), &pluginsdk.WebhookRequest{WebhookKey: "oauth-callback", Method: "GET", Query: "state=" + url.QueryEscape(state) + "&code=oauth-code"})
	require.NoError(t, err)
	require.Equal(t, int32(http.StatusOK), response.Status)
	response, err = workflows.HandleWebhook(context.Background(), &pluginsdk.WebhookRequest{WebhookKey: "oauth-callback", Method: "GET", Query: "state=" + url.QueryEscape(state) + "&code=oauth-code"})
	require.NoError(t, err)
	require.Equal(t, int32(http.StatusBadRequest), response.Status)

	provider, err := resolver.Provider(context.Background(), "workspace-1")
	require.NoError(t, err)
	require.NoError(t, provider.Health(context.Background()))
	credential, err := provider.ResolveGitCredential(context.Background())
	require.NoError(t, err)
	require.Equal(t, "x-token-auth", credential.Username)
	require.Equal(t, "access-token", credential.Secret)
	require.NotContains(t, flatten(host.state), "access-token")
	require.NotContains(t, flatten(host.state), "refresh-token")
}

func TestWorkflows_ConnectionSaveRejectsOAuthEndpointOverrides(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)

	_, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
		ActionKey: "connection.save", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
		Body: []byte(`{"product":"cloud","cloud_workspace":"acme","auth_method":"oauth","oauth_client_id":"client-id","oauth_client_secret":"client-secret","oauth_redirect_url":"https://plugin.example.test/callback","oauth_authorization_url":"https://attacker.example.test/authorize"}`),
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_DerivesDataCenterOAuthEndpointsAndScopes(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test/bitbucket", AuthMethod: "oauth", AuthIdentity: "dev",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)

	started, err := resolver.StartOAuth(context.Background(), "workspace-1")
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.example.test/bitbucket/rest/oauth2/latest/authorize", started.Scheme+"://"+started.Host+started.Path)
	require.Equal(t, "REPO_READ REPO_WRITE", started.Query().Get("scope"))

	settings, found, err := resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, err)
	require.True(t, found)
	registration, err := resolver.oauthRegistration(context.Background(), "workspace-1", settings)
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.example.test/bitbucket/rest/oauth2/latest/token", registration.TokenURL.String())
}

func TestConnectionResolver_UsesFixedCloudOAuthEndpointAndScopes(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)

	started, err := resolver.StartOAuth(context.Background(), "workspace-1")
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.org/site/oauth2/authorize", started.Scheme+"://"+started.Host+started.Path)
	require.Equal(t, "account repository repository:write pullrequest pullrequest:write", started.Query().Get("scope"))
}

func TestWorkflows_OAuthCallbackWithoutCodeFailsWithoutGrant(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	started, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "oauth.start", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}})
	require.NoError(t, err)
	var authorization struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(started.Body, &authorization))
	parsed, err := url.Parse(authorization.URL)
	require.NoError(t, err)

	response, err := workflows.HandleWebhook(context.Background(), &pluginsdk.WebhookRequest{WebhookKey: "oauth-callback", Method: "GET", Query: "state=" + url.QueryEscape(parsed.Query().Get("state"))})
	require.NoError(t, err)
	require.Equal(t, int32(http.StatusBadRequest), response.Status)
	_, grantFound := host.secrets[oauthCredentialSecretKey("workspace-1", 1)]
	require.False(t, grantFound)
}

func TestWorkflows_ConnectionGetNeverReflectsOAuthSecrets(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/2.0/user", request.URL.Path)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	resolver.cloudAPIBaseOverride, err = url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	host.secrets[oauthStateSecretKey("workspace-1")] = "state-key"
	require.NoError(t, (hostOAuthCredentialRepository{host: host}).SaveCredential(context.Background(), auth.CredentialScope{WorkspaceID: "workspace-1", Generation: settings.OAuthGeneration}, auth.Credential{
		AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Now().Add(time.Hour),
	}))
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "connection.get", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}})
	require.NoError(t, err)
	for _, secret := range []string{"client-secret", "access-token", "refresh-token", "state-key"} {
		require.NotContains(t, string(response.Body), secret)
	}
	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body, &payload))
	require.Equal(t, true, payload["oauth_registration_configured"])
	require.NotContains(t, payload, "oauth_authorization_url")
	require.NotContains(t, payload, "oauth_token_url")
}

func TestWorkflows_UnconfiguredWorkspaceReturnsEmptyBrowseSurfaces(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)

	tests := []struct {
		action string
		want   string
	}{
		{action: "repositories.list", want: `{"repositories":[]}`},
		{action: "pullrequests.queue", want: `{"pull_requests":[]}`},
	}
	for _, test := range tests {
		t.Run(test.action, func(t *testing.T) {
			response, actionErr := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{
				ActionKey: test.action,
				Context:   pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"},
			})
			require.NoError(t, actionErr)
			require.JSONEq(t, test.want, string(response.Body))
		})
	}
}

func TestWorkflows_ConnectionDisconnectRevokesEveryCredential(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "api-token",
	})
	require.NoError(t, err)
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	require.NoError(t, (hostOAuthCredentialRepository{host: host}).SaveCredential(context.Background(), auth.CredentialScope{WorkspaceID: "workspace-1", Generation: settings.OAuthGeneration}, auth.Credential{
		AccessToken: "old-access", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(time.Hour),
	}))
	workflows, err := NewWorkflows(host, resolver)
	require.NoError(t, err)
	started, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "oauth.start", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}})
	require.NoError(t, err)
	var authorization struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(started.Body, &authorization))
	authorizationURL, err := url.Parse(authorization.URL)
	require.NoError(t, err)
	state := authorizationURL.Query().Get("state")
	require.NotEmpty(t, state)
	provider, err := resolver.Provider(context.Background(), "workspace-1")
	require.NoError(t, err)
	credential, err := provider.ResolveGitCredential(context.Background())
	require.NoError(t, err)
	require.Equal(t, "old-access", credential.Secret)

	response, err := workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "connection.disconnect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}})
	require.NoError(t, err)
	require.JSONEq(t, `{"state":"unconfigured","healthy":false}`, string(response.Body))
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
	callback, err := workflows.HandleWebhook(context.Background(), &pluginsdk.WebhookRequest{WebhookKey: "oauth-callback", Method: "GET", Query: "state=" + url.QueryEscape(state) + "&code=old-code"})
	require.NoError(t, err)
	require.Equal(t, int32(http.StatusBadRequest), callback.Status, "pending OAuth callback must be revoked")
	_, err = provider.ResolveGitCredential(context.Background())
	require.Error(t, err, "already-resolved provider must not reuse revoked OAuth credential")
	_, err = resolver.Provider(context.Background(), "workspace-1")
	require.Error(t, err, "broker re-resolution must fail closed")
	_, err = workflows.ResolveGitCredential(context.Background(), &pluginsdk.ResolveGitCredentialRequest{
		ProviderID: "bitbucket", WorkspaceID: "workspace-1", TaskID: "task-1", SessionID: "session-1", RepositoryID: "repository-1", Host: "bitbucket.org", Path: "/acme/repo.git",
	})
	require.ErrorIs(t, err, ErrCredentialUnavailable)

	response, err = workflows.HandleAction(context.Background(), &pluginsdk.PluginActionRequest{ActionKey: "connection.disconnect", Context: pluginsdk.VerifiedActionContext{WorkspaceID: "workspace-1"}})
	require.NoError(t, err, "disconnect is idempotent")
	require.JSONEq(t, `{"state":"unconfigured","healthy":false}`, string(response.Body))
}

func TestConnectionResolver_DisconnectRetriesSecretRevocationAfterFailure(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	settings, err := resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token",
		AuthIdentity: "dev@example.test", Token: "api-token",
	})
	require.NoError(t, err)
	tokenKey := connectionSecretKey("workspace-1", settings.CredentialGeneration)
	host.deleteSecretErrKey = tokenKey

	err = resolver.Disconnect(context.Background(), "workspace-1")
	require.ErrorContains(t, err, "delete Bitbucket credential")
	_, found, loadErr := resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, loadErr)
	require.False(t, found, "a failed secret cleanup must still disable the connection")
	require.Equal(t, "api-token", host.secrets[tokenKey])

	host.deleteSecretErrKey = ""
	require.NoError(t, resolver.Disconnect(context.Background(), "workspace-1"))
	_, tokenFound := host.secrets[tokenKey]
	require.False(t, tokenFound, "a retry must retain enough metadata to revoke the orphaned generation")
	require.Empty(t, host.state)
}

func TestConnectionResolver_DisconnectPersistsLegacyCleanupWithoutConnectionState(t *testing.T) {
	host := newConnectionHost()
	legacyKey := legacyConnectionSecretKey("workspace-1")
	host.secrets[legacyKey] = "legacy-token"
	host.deleteSecretErrKey = legacyKey
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	err = resolver.Disconnect(context.Background(), "workspace-1")
	require.ErrorContains(t, err, "delete Bitbucket credential")
	_, found, loadErr := resolver.Load(context.Background(), "workspace-1")
	require.NoError(t, loadErr)
	require.False(t, found)
	require.NotEmpty(t, host.state, "failed cleanup needs a retryable tombstone")

	host.deleteSecretErrKey = ""
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "api_token",
		AuthIdentity: "dev@example.test", Token: "new-token",
	})
	require.NoError(t, err)
	_, legacyFound := host.secrets[legacyKey]
	require.False(t, legacyFound)
}

func TestConnectionResolver_BoundsPendingOAuthFlows(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)

	for range 36 {
		_, err = resolver.StartOAuth(context.Background(), "workspace-1")
		require.NoError(t, err)
	}
	require.LessOrEqual(t, len(resolver.oauthFlows), 32)
}

func TestConnectionResolver_RejectsExpiredOAuthFlow(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	started, err := resolver.StartOAuth(context.Background(), "workspace-1")
	require.NoError(t, err)
	state := started.Query().Get("state")
	require.NotEmpty(t, state)
	resolver.oauthMu.Lock()
	flow := resolver.oauthFlows[state]
	flow.expiresAt = time.Now().Add(-time.Second)
	resolver.oauthFlows[state] = flow
	resolver.oauthMu.Unlock()

	_, err = resolver.HandleOAuthCallback(context.Background(), state, "code")
	require.Error(t, err)
	require.Empty(t, resolver.oauthFlows)
}

func TestConnectionResolver_AllowsOnlyVerifiedForkSourceCredentialScope(t *testing.T) {
	base := newConnectionHost()
	host := &scopedConnectionHost{
		connectionHost: base,
		tasks: &taskReader{task: &pluginsdk.Task{
			ID: "task-1", WorkspaceID: "workspace-1", Repositories: []pluginsdk.TaskRepository{{RepositoryID: "fork-repository", CheckoutBranch: "feature"}},
		}},
		repositories: &repositoryReader{repositories: []pluginsdk.Repository{{
			ID: "fork-repository", WorkspaceID: "workspace-1", SourceType: "provider", ProviderID: "bitbucket", ProviderHost: "https://bitbucket.org",
			OwnerOrProject: "fork", ProviderRepositoryID: "fork-uuid", ProviderName: "repo", Name: "repo", RemoteURL: "https://bitbucket.org/fork/repo.git",
		}}},
	}
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{Product: domain.ProductCloud, CloudWorkspace: "destination", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "token"})
	require.NoError(t, err)

	scope := GitCredentialScope{WorkspaceID: "workspace-1", TaskID: "task-1", RepositoryID: "fork-repository", Host: "bitbucket.org", Path: "/fork/repo.git"}
	require.NoError(t, resolver.ValidateGitCredentialScope(context.Background(), scope))
	binding, err := resolver.GitCredentialBinding(context.Background(), scope)
	require.NoError(t, err)
	require.NotEmpty(t, binding)
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{Product: domain.ProductCloud, CloudWorkspace: "destination", AuthMethod: "api_token", AuthIdentity: "dev@example.test", Token: "rotated-token"})
	require.NoError(t, err)
	rotatedBinding, err := resolver.GitCredentialBinding(context.Background(), scope)
	require.NoError(t, err)
	require.NotEqual(t, binding, rotatedBinding)
	require.NoError(t, resolver.Disconnect(context.Background(), "workspace-1"))
	_, err = resolver.GitCredentialBinding(context.Background(), scope)
	require.ErrorIs(t, err, ErrCredentialUnavailable)
	scope.Path = "/fork/other.git"
	require.ErrorIs(t, resolver.ValidateGitCredentialScope(context.Background(), scope), ErrCredentialUnavailable)
	scope.Path = "/fork/repo.git"
	host.repositories.repositories[0].ProviderName = "other"
	require.ErrorIs(t, resolver.ValidateGitCredentialScope(context.Background(), scope), ErrCredentialUnavailable)
	host.repositories.repositories[0].ProviderName = "repo"
	host.repositories.repositories[0].ProviderHost = "https://user@bitbucket.org"
	require.ErrorIs(t, resolver.ValidateGitCredentialScope(context.Background(), scope), ErrCredentialUnavailable)
	host.repositories.repositories[0].ProviderHost = "https://bitbucket.org/path"
	require.ErrorIs(t, resolver.ValidateGitCredentialScope(context.Background(), scope), ErrCredentialUnavailable)
	host.repositories.repositories[0].ProviderHost = "https://bitbucket.org"
	host.repositories.repositories[0].RemoteURL = "https://bitbucket.org/fork/repo.git?credential=leak"
	require.ErrorIs(t, resolver.ValidateGitCredentialScope(context.Background(), scope), ErrCredentialUnavailable)
}

func TestConnectionResolver_UsesDataCenterProjectTokenBearerAndGitTokenUser(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "Bearer project-token", request.Header.Get("Authorization"))
		require.Equal(t, "/rest/api/latest/repos", request.URL.Path)
		require.Equal(t, "widget", request.URL.Query().Get("name"))
		_, _ = writer.Write([]byte(`{"values":[],"isLastPage":true}`))
	}))
	defer server.Close()

	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: server.URL, AuthMethod: "project_token", Token: "project-token",
	})
	require.NoError(t, err)
	provider, err := resolver.Provider(context.Background(), "workspace-1")
	require.NoError(t, err)
	_, err = provider.ListRepositories(context.Background(), "widget", 1)
	require.NoError(t, err)
	credential, err := provider.ResolveGitCredential(context.Background())
	require.NoError(t, err)
	require.Equal(t, "x-token-auth", credential.Username)
}

func TestDataCenterAuthenticationKeepsScopedTokenMode(t *testing.T) {
	project, err := dataCenterAuthentication(ConnectionSettings{Product: domain.ProductDataCenter, AuthMethod: "project_token"})
	require.NoError(t, err)
	require.Equal(t, datacenter.AuthenticationProjectToken, project.Mode)

	repository, err := dataCenterAuthentication(ConnectionSettings{Product: domain.ProductDataCenter, AuthMethod: "repository_token"})
	require.NoError(t, err)
	require.Equal(t, datacenter.AuthenticationRepositoryToken, repository.Mode)
}

func TestConnectionResolver_UsesDataCenterUserPATBasicAndIdentity(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, token, ok := request.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "dev", username)
		require.Equal(t, "pat-token", token)
		_, _ = writer.Write([]byte(`{"values":[],"isLastPage":true}`))
	}))
	defer server.Close()

	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)
	resolver.httpClient = server.Client()
	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: server.URL, AuthMethod: "user_pat", AuthIdentity: "dev", Token: "pat-token",
	})
	require.NoError(t, err)
	provider, err := resolver.Provider(context.Background(), "workspace-1")
	require.NoError(t, err)
	_, err = provider.ListRepositories(context.Background(), "", 1)
	require.NoError(t, err)
	credential, err := provider.ResolveGitCredential(context.Background())
	require.NoError(t, err)
	require.Equal(t, "dev", credential.Username)
}

func TestConnectionResolver_RejectsCloudAPITokenModeForDataCenter(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "api_token", AuthIdentity: "dev", Token: "pat-token",
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_RejectsCloudConnectionWithoutExplicitWorkspace(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "kandev-workspace-id", ConnectionInput{Product: domain.ProductCloud, Token: "token"})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_RejectsDataCenterOAuthWithoutGitUsername(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "oauth",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

func TestConnectionResolver_AcceptsDataCenterOAuthWithoutExplicitEndpoints(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductDataCenter, BaseURL: "https://bitbucket.example.test", AuthMethod: "oauth", AuthIdentity: "dev",
		OAuthClientID: "client-id", OAuthClientSecret: "client-secret", OAuthRedirectURL: "https://plugin.example.test/callback",
	})
	require.NoError(t, err)
	require.NotEmpty(t, host.state)
	require.NotEmpty(t, host.secrets)
}

func TestConnectionResolver_RejectsConnectionWithoutAuthMethod(t *testing.T) {
	host := newConnectionHost()
	resolver, err := NewConnectionResolver(host)
	require.NoError(t, err)

	_, err = resolver.Save(context.Background(), "workspace-1", ConnectionInput{
		Product: domain.ProductCloud, CloudWorkspace: "acme", AuthIdentity: "dev@example.test", Token: "token",
	})
	require.Error(t, err)
	require.Empty(t, host.state)
	require.Empty(t, host.secrets)
}

type connectionHost struct {
	pluginsdk.UnimplementedHostData
	state               map[string]map[string]any
	secrets             map[string]string
	setStateErr         error
	deleteSecretErrKey  string
	beforeSetStateError func()
	deletedSecretKeys   []string
}

type scopedConnectionHost struct {
	*connectionHost
	tasks        *taskReader
	repositories *repositoryReader
}

func (h *scopedConnectionHost) Tasks() pluginsdk.TaskReader              { return h.tasks }
func (h *scopedConnectionHost) Repositories() pluginsdk.RepositoryReader { return h.repositories }

func newConnectionHost() *connectionHost {
	return &connectionHost{state: make(map[string]map[string]any), secrets: make(map[string]string)}
}

func (h *connectionHost) GetState(_ context.Context, scope, scopeID, key string) (map[string]any, bool, error) {
	value, found := h.state[scope+":"+scopeID+":"+key]
	return value, found, nil
}
func (h *connectionHost) SetState(_ context.Context, scope, scopeID, key string, value map[string]any) error {
	if h.setStateErr != nil {
		if h.beforeSetStateError != nil {
			h.beforeSetStateError()
		}
		return h.setStateErr
	}
	h.state[scope+":"+scopeID+":"+key] = value
	return nil
}
func (h *connectionHost) DeleteState(_ context.Context, scope, scopeID, key string) error {
	delete(h.state, scope+":"+scopeID+":"+key)
	return nil
}
func (*connectionHost) ListState(context.Context, string, string) ([]pluginsdk.StateEntry, error) {
	return nil, nil
}
func (*connectionHost) GetConfig(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (*connectionHost) RevealSecret(context.Context, string) (string, error) { return "", nil }
func (h *connectionHost) GetSecret(_ context.Context, key string) (string, bool, error) {
	value, found := h.secrets[key]
	return value, found, nil
}
func (h *connectionHost) SetSecret(_ context.Context, key, value string) error {
	h.secrets[key] = value
	return nil
}
func (h *connectionHost) DeleteSecret(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if key == h.deleteSecretErrKey {
		return errors.New("secret deletion unavailable")
	}
	h.deletedSecretKeys = append(h.deletedSecretKeys, key)
	delete(h.secrets, key)
	return nil
}
func (*connectionHost) EmitEvent(context.Context, string, map[string]any) error { return nil }

func flatten(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
