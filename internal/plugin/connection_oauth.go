package plugin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"kandev-plugin-bitbucket/internal/auth"
	"kandev-plugin-bitbucket/internal/datacenter"
	"kandev-plugin-bitbucket/internal/domain"

	"github.com/kandev/kandev/pkg/pluginsdk"
)

const maxOAuthFlowsPerWorkspace = 32

type oauthEndpointPair struct {
	authorization *url.URL
	token         *url.URL
}

type oauthFlow struct {
	registration auth.OAuthRegistration
	scope        auth.CredentialScope
	states       *auth.StateManager
	epoch        uint64
	expiresAt    time.Time
}

// StartOAuth begins a BYO OAuth authorization-code flow. Credentials
// are exchanged only by the callback and are kept in host secret storage.
func (r *ConnectionResolver) StartOAuth(ctx context.Context, workspaceID string) (*url.URL, error) {
	settings, found, err := r.Load(ctx, workspaceID)
	if err != nil {
		return nil, unavailableActionError("load Bitbucket OAuth connection: %v", err)
	}
	if !found || settings.AuthMethod != "oauth" {
		return nil, conflictActionError("Bitbucket OAuth is not configured")
	}
	registration, err := r.oauthRegistration(ctx, workspaceID, settings)
	if err != nil {
		return nil, err
	}
	states, err := r.oauthStateManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	scope := oauthCredentialScope(workspaceID, settings)
	r.oauthMu.Lock()
	epoch := r.oauthEpoch[workspaceID]
	r.oauthMu.Unlock()
	request, err := auth.StartAuthorization(ctx, states, scope, registration, oauthScopes(settings.Product))
	if err != nil {
		return nil, err
	}
	r.oauthMu.Lock()
	if epoch != r.oauthEpoch[workspaceID] {
		r.oauthMu.Unlock()
		return nil, fmt.Errorf("Bitbucket OAuth connection was revoked")
	}
	r.purgeOAuthFlowsLocked(time.Now())
	r.evictOAuthFlowLocked(workspaceID)
	r.oauthFlows[request.State] = oauthFlow{registration: registration, scope: scope, states: states, epoch: epoch, expiresAt: request.ExpiresAt}
	r.oauthMu.Unlock()
	return request.URL, nil
}

// HandleOAuthCallback validates and consumes a PKCE state exactly once before
// exchanging the code and saving a generation-bound rotating credential.
func (r *ConnectionResolver) HandleOAuthCallback(ctx context.Context, state, code string) (*pluginsdk.WebhookResponse, error) {
	r.oauthMu.Lock()
	flow, found := r.oauthFlows[state]
	if found {
		delete(r.oauthFlows, state)
	}
	if found && (flow.epoch != r.oauthEpoch[flow.scope.WorkspaceID] || !flow.expiresAt.After(time.Now())) {
		found = false
	}
	r.oauthMu.Unlock()
	if !found || flow.states == nil || code == "" {
		return nil, fmt.Errorf("invalid Bitbucket OAuth callback")
	}
	credential, err := r.oauthRefresher().ExchangeAuthorizationCode(ctx, flow.states, state, code, flow.registration)
	if err != nil {
		return nil, fmt.Errorf("complete Bitbucket OAuth: %w", err)
	}
	r.oauthMu.Lock()
	defer r.oauthMu.Unlock()
	if flow.epoch != r.oauthEpoch[flow.scope.WorkspaceID] {
		return nil, fmt.Errorf("Bitbucket OAuth connection was revoked")
	}
	if err := (hostOAuthCredentialRepository{host: r.host}).SaveCredential(ctx, flow.scope, credential); err != nil {
		return nil, err
	}
	return &pluginsdk.WebhookResponse{Status: http.StatusOK, Headers: map[string]string{"Content-Type": "text/plain; charset=utf-8"}, Body: []byte("Bitbucket connected. You can close this window.")}, nil
}

func (r *ConnectionResolver) configureOAuthSettings(
	ctx context.Context,
	workspaceID string,
	settings *ConnectionSettings,
	previous ConnectionSettings,
	found bool,
	input ConnectionInput,
) error {
	clientID := strings.TrimSpace(input.OAuthClientID)
	clientSecret := strings.TrimSpace(input.OAuthClientSecret)
	redirectURL := strings.TrimSpace(input.OAuthRedirectURL)
	canReuse := found && previous.AuthMethod == "oauth" && previous.Product == settings.Product && clientSecret == ""
	if canReuse {
		if (clientID != "" && clientID != previous.OAuthClientID) || (redirectURL != "" && redirectURL != previous.OAuthRedirectURL) {
			return invalidConnectionInput("Bitbucket OAuth client secret is required when registration changes")
		}
		configured, err := r.oauthRegistrationConfiguredForGeneration(
			ctx,
			workspaceID,
			previous.OAuthGeneration,
		)
		if err != nil {
			return err
		}
		if !configured {
			return invalidConnectionInput("Bitbucket OAuth client registration is required")
		}
		settings.OAuthClientID = previous.OAuthClientID
		settings.OAuthRedirectURL = previous.OAuthRedirectURL
		settings.OAuthGeneration = previous.OAuthGeneration
		return nil
	}
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return invalidConnectionInput("Bitbucket OAuth client id, client secret, and redirect URL are required")
	}
	settings.OAuthClientID = clientID
	settings.OAuthRedirectURL = redirectURL
	if err := validateOAuthRedirectURL(settings.OAuthRedirectURL, "Bitbucket OAuth redirect URL"); err != nil {
		return invalidConnectionInput("%v", err)
	}
	settings.OAuthGeneration = previous.OAuthGeneration + 1
	if settings.OAuthGeneration == 0 {
		settings.OAuthGeneration = 1
	}
	return nil
}

func (r *ConnectionResolver) oauthRegistrationConfigured(ctx context.Context, workspaceID string) (bool, error) {
	settings, found, err := r.Load(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	generation := uint64(0)
	if found {
		generation = settings.OAuthGeneration
	}
	return r.oauthRegistrationConfiguredForGeneration(ctx, workspaceID, generation)
}

func (r *ConnectionResolver) oauthRegistrationConfiguredForGeneration(
	ctx context.Context,
	workspaceID string,
	generation uint64,
) (bool, error) {
	secret, found, err := r.loadOAuthRegistrationSecret(ctx, workspaceID, generation)
	if err != nil {
		return false, fmt.Errorf("load Bitbucket OAuth client registration: %w", err)
	}
	if !found {
		return false, nil
	}
	var registration oauthRegistrationSecret
	if err := json.Unmarshal([]byte(secret), &registration); err != nil {
		return false, nil
	}
	return strings.TrimSpace(registration.ClientSecret) != "", nil
}

func (r *ConnectionResolver) invalidateSupersededOAuth(workspaceID string, previous ConnectionSettings, found bool, next ConnectionSettings) {
	if !found {
		return
	}
	previousOAuth := previous.AuthMethod == "oauth"
	nextOAuth := next.AuthMethod == "oauth"
	if previousOAuth && (!nextOAuth || previous.OAuthGeneration != next.OAuthGeneration || previous.ConnectionBinding != next.ConnectionBinding) {
		r.invalidateOAuthWorkspace(workspaceID, previous, true)
	}
}

func (r *ConnectionResolver) invalidateOAuthWorkspace(workspaceID string, settings ConnectionSettings, hasSettings bool) {
	r.oauthMu.Lock()
	r.oauthEpoch[workspaceID]++
	delete(r.oauthState, workspaceID)
	for state, flow := range r.oauthFlows {
		if flow.scope.WorkspaceID == workspaceID {
			delete(r.oauthFlows, state)
		}
	}
	r.oauthMu.Unlock()
	if !hasSettings || settings.AuthMethod != "oauth" || settings.OAuthGeneration == 0 {
		return
	}
	r.refresherMu.Lock()
	refresher := r.refresher
	r.refresherMu.Unlock()
	if refresher != nil {
		refresher.Invalidate(oauthCredentialScope(workspaceID, settings))
	}
}

func oauthCredentialScope(workspaceID string, settings ConnectionSettings) auth.CredentialScope {
	return auth.CredentialScope{
		WorkspaceID: workspaceID, Generation: settings.OAuthGeneration,
		ConnectionBinding: settings.ConnectionBinding,
	}
}

func (r *ConnectionResolver) purgeOAuthFlowsLocked(now time.Time) {
	for state, flow := range r.oauthFlows {
		if !flow.expiresAt.After(now) {
			delete(r.oauthFlows, state)
		}
	}
}

func (r *ConnectionResolver) evictOAuthFlowLocked(workspaceID string) {
	count := 0
	var oldestState string
	var oldest time.Time
	for state, flow := range r.oauthFlows {
		if flow.scope.WorkspaceID != workspaceID {
			continue
		}
		count++
		if oldestState == "" || flow.expiresAt.Before(oldest) {
			oldestState = state
			oldest = flow.expiresAt
		}
	}
	if count >= maxOAuthFlowsPerWorkspace && oldestState != "" {
		delete(r.oauthFlows, oldestState)
	}
}

func (r *ConnectionResolver) oauthEndpoints(settings ConnectionSettings) (*url.URL, *url.URL, error) {
	if override := r.oauthEndpointOverride; override != nil {
		if override.authorization == nil || override.token == nil {
			return nil, nil, fmt.Errorf("OAuth endpoint test override is incomplete")
		}
		if err := validateHTTPSURL(override.authorization.String(), "Bitbucket OAuth authorization URL"); err != nil {
			return nil, nil, err
		}
		if err := validateHTTPSURL(override.token.String(), "Bitbucket OAuth token URL"); err != nil {
			return nil, nil, err
		}
		authorization := *override.authorization
		token := *override.token
		return &authorization, &token, nil
	}
	if settings.Product == domain.ProductCloud {
		authorization, _ := url.Parse("https://bitbucket.org/site/oauth2/authorize")
		token, _ := url.Parse("https://bitbucket.org/site/oauth2/access_token")
		return authorization, token, nil
	}
	if settings.Product != domain.ProductDataCenter {
		return nil, nil, fmt.Errorf("unsupported Bitbucket product")
	}
	if _, err := datacenter.NewConnection(datacenter.ConnectionOptions{BaseURL: settings.BaseURL}); err != nil {
		return nil, nil, err
	}
	base, err := url.Parse(settings.BaseURL)
	if err != nil {
		return nil, nil, err
	}
	base.Path = strings.TrimSuffix(path.Clean("/"+strings.TrimPrefix(base.Path, "/")), "/")
	if base.Path == "." || base.Path == "/" {
		base.Path = ""
	}
	base.RawPath = ""
	authorization := *base
	authorization.Path = path.Join(base.Path, "rest", "oauth2", "latest", "authorize")
	token := *base
	token.Path = path.Join(base.Path, "rest", "oauth2", "latest", "token")
	return &authorization, &token, nil
}

func (r *ConnectionResolver) oauthStateManager(ctx context.Context, workspaceID string) (*auth.StateManager, error) {
	r.oauthMu.Lock()
	defer r.oauthMu.Unlock()
	if manager := r.oauthState[workspaceID]; manager != nil {
		return manager, nil
	}
	key, found, err := r.host.GetSecret(ctx, oauthStateSecretKey(workspaceID))
	if err != nil {
		return nil, err
	}
	if !found {
		value := make([]byte, 32)
		if _, err := rand.Read(value); err != nil {
			return nil, err
		}
		key = base64.RawURLEncoding.EncodeToString(value)
		if err := r.host.SetSecret(ctx, oauthStateSecretKey(workspaceID), key); err != nil {
			return nil, err
		}
	}
	manager, err := auth.NewStateManager([]byte(key), auth.NewMemoryPendingStore(), time.Now)
	if err != nil {
		return nil, err
	}
	r.oauthState[workspaceID] = manager
	return manager, nil
}
