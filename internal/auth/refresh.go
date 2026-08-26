package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxTokenResponseBytes int64 = 1 << 20

// OAuthRegistration is supplied by each workspace's own OAuth client.
type OAuthRegistration struct {
	ClientID         string
	ClientSecret     string
	AuthorizationURL *url.URL
	TokenURL         *url.URL
	RedirectURL      *url.URL
}

// CredentialScope keys credentials to a workspace and its current generation.
type CredentialScope struct {
	WorkspaceID       string
	Generation        uint64
	ConnectionBinding string
}

// ErrCredentialRevoked prevents a superseded connection from publishing a
// refresh result after its credential generation has been invalidated.
var ErrCredentialRevoked = errors.New("OAuth credential generation was revoked")

// Credential contains an in-memory OAuth access/refresh token pair.
type Credential struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// AuthorizationRequest is the browser redirect derived from a server-side PKCE flow.
type AuthorizationRequest struct {
	URL       *url.URL
	State     string
	ExpiresAt time.Time
}

// StartAuthorization creates an authorization-code redirect for a BYO OAuth client.
func StartAuthorization(ctx context.Context, states *StateManager, scope CredentialScope, registration OAuthRegistration, scopes []string) (AuthorizationRequest, error) {
	if states == nil {
		return AuthorizationRequest{}, fmt.Errorf("OAuth state manager is required")
	}
	if registration.ClientID == "" || registration.AuthorizationURL == nil || registration.RedirectURL == nil {
		return AuthorizationRequest{}, fmt.Errorf("OAuth client registration is incomplete")
	}
	if err := validateHTTPSURL(registration.AuthorizationURL, "authorization endpoint"); err != nil {
		return AuthorizationRequest{}, err
	}
	if err := ValidateOAuthRedirectURL(registration.RedirectURL); err != nil {
		return AuthorizationRequest{}, err
	}
	pending, err := states.Start(ctx, scope.WorkspaceID, scope.Generation)
	if err != nil {
		return AuthorizationRequest{}, err
	}
	authorizationURL := *registration.AuthorizationURL
	query := authorizationURL.Query()
	query.Set("client_id", registration.ClientID)
	query.Set("response_type", "code")
	query.Set("redirect_uri", registration.RedirectURL.String())
	query.Set("state", pending.State)
	query.Set("code_challenge", pending.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	if len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}
	authorizationURL.RawQuery = query.Encode()
	return AuthorizationRequest{URL: &authorizationURL, State: pending.State, ExpiresAt: pending.ExpiresAt}, nil
}

// Refresher coalesces refresh requests for one workspace credential generation.
type Refresher struct {
	httpClient *http.Client
	now        func() time.Time

	mu        sync.Mutex
	inFlight  map[string]*refreshCall
	completed map[string]completedRefresh
	revoked   map[string]struct{}
}

type refreshCall struct {
	done       chan struct{}
	credential Credential
	err        error
}

type completedRefresh struct {
	generation   uint64
	refreshToken string
	credential   Credential
}

// NewRefresher creates a refresh coordinator.
func NewRefresher(httpClient *http.Client, now func() time.Time) *Refresher {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	safeClient := *httpClient
	safeClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	if now == nil {
		now = time.Now
	}
	return &Refresher{
		httpClient: &safeClient,
		now:        now,
		inFlight:   make(map[string]*refreshCall),
		completed:  make(map[string]completedRefresh),
		revoked:    make(map[string]struct{}),
	}
}

// Invalidate permanently revokes one connection-bound credential generation
// for this process. The non-secret tombstone also fences in-flight exchanges.
func (r *Refresher) Invalidate(scope CredentialScope) {
	if r == nil || scope.WorkspaceID == "" || scope.Generation == 0 {
		return
	}
	key := credentialScopeKey(scope)
	r.mu.Lock()
	delete(r.completed, key)
	r.revoked[key] = struct{}{}
	r.mu.Unlock()
}

// Refresh exchanges a rotating refresh token once per workspace generation.
func (r *Refresher) Refresh(ctx context.Context, scope CredentialScope, registration OAuthRegistration, refreshToken string) (Credential, error) {
	if scope.WorkspaceID == "" || scope.Generation == 0 {
		return Credential{}, fmt.Errorf("workspace and credential generation are required")
	}
	if refreshToken == "" {
		return Credential{}, fmt.Errorf("refresh token is required")
	}
	key := credentialScopeKey(scope)
	r.mu.Lock()
	r.purgeExpiredCompletedLocked(r.now())
	if _, revoked := r.revoked[key]; revoked {
		r.mu.Unlock()
		return Credential{}, ErrCredentialRevoked
	}
	if completed := r.completed[key]; completed.generation == scope.Generation &&
		completed.refreshToken == refreshToken && completed.credential.ExpiresAt.After(r.now()) {
		r.mu.Unlock()
		return completed.credential, nil
	}
	if call := r.inFlight[key]; call != nil {
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return Credential{}, ctx.Err()
		case <-call.done:
			return call.credential, call.err
		}
	}
	call := &refreshCall{done: make(chan struct{})}
	r.inFlight[key] = call
	r.mu.Unlock()

	credential, err := r.exchange(ctx, registration, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
	r.mu.Lock()
	if _, revoked := r.revoked[key]; revoked {
		credential = Credential{}
		err = ErrCredentialRevoked
	}
	call.credential = credential
	call.err = err
	if err == nil {
		r.completed[key] = completedRefresh{
			generation:   scope.Generation,
			refreshToken: refreshToken,
			credential:   credential,
		}
	}
	delete(r.inFlight, key)
	close(call.done)
	r.mu.Unlock()
	return credential, err
}

func credentialScopeKey(scope CredentialScope) string {
	return scope.WorkspaceID + "\x00" + fmt.Sprint(scope.Generation) + "\x00" + scope.ConnectionBinding
}

func (r *Refresher) purgeExpiredCompletedLocked(now time.Time) {
	for key, completed := range r.completed {
		if !completed.credential.ExpiresAt.After(now) {
			delete(r.completed, key)
		}
	}
}

// ExchangeAuthorizationCode consumes a one-time PKCE state before token exchange.
func (r *Refresher) ExchangeAuthorizationCode(ctx context.Context, states *StateManager, state, code string, registration OAuthRegistration) (Credential, error) {
	if states == nil || code == "" {
		return Credential{}, fmt.Errorf("OAuth callback is incomplete")
	}
	pending, err := states.Consume(ctx, state)
	if err != nil {
		return Credential{}, err
	}
	if registration.RedirectURL == nil {
		return Credential{}, fmt.Errorf("OAuth client registration is incomplete")
	}
	return r.exchange(ctx, registration, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {registration.RedirectURL.String()},
		"code_verifier": {pending.CodeVerifier},
	})
}

func (r *Refresher) exchange(ctx context.Context, registration OAuthRegistration, values url.Values) (Credential, error) {
	if registration.ClientID == "" || registration.ClientSecret == "" || registration.TokenURL == nil {
		return Credential{}, fmt.Errorf("OAuth client registration is incomplete")
	}
	if err := validateTokenURL(registration.TokenURL); err != nil {
		return Credential{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, registration.TokenURL.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return Credential{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(registration.ClientID, registration.ClientSecret)
	response, err := r.httpClient.Do(request)
	if err != nil {
		return Credential{}, fmt.Errorf("exchange OAuth credential: %w", err)
	}
	defer response.Body.Close()
	body, err := readBounded(response.Body, maxTokenResponseBytes)
	if err != nil {
		return Credential{}, err
	}
	if response.StatusCode != http.StatusOK {
		return Credential{}, fmt.Errorf("OAuth token endpoint returned %s", response.Status)
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Credential{}, fmt.Errorf("decode OAuth token response: %w", err)
	}
	if payload.AccessToken == "" || payload.RefreshToken == "" || payload.ExpiresIn <= 0 {
		return Credential{}, fmt.Errorf("OAuth token response is incomplete")
	}
	return Credential{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresAt:    r.now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func validateTokenURL(tokenURL *url.URL) error {
	if tokenURL.User != nil || tokenURL.Hostname() == "" || tokenURL.RawQuery != "" || tokenURL.Fragment != "" {
		return fmt.Errorf("OAuth token endpoint is invalid")
	}
	if tokenURL.Scheme == "https" {
		return nil
	}
	if tokenURL.Scheme == "http" && net.ParseIP(tokenURL.Hostname()).IsLoopback() {
		return nil
	}
	return fmt.Errorf("OAuth token endpoint must use HTTPS")
}

func validateHTTPSURL(value *url.URL, name string) error {
	if value.User != nil || value.Hostname() == "" || value.RawQuery != "" || value.Fragment != "" || value.Scheme != "https" {
		return fmt.Errorf("OAuth %s must use HTTPS without credentials, query, or fragment", name)
	}
	return nil
}

// ValidateOAuthRedirectURL validates the callback URL used by an OAuth flow.
func ValidateOAuthRedirectURL(value *url.URL) error {
	if value == nil || value.User != nil || value.Hostname() == "" || value.RawQuery != "" || value.Fragment != "" {
		return fmt.Errorf("OAuth redirect URL must not contain credentials, a query, or a fragment")
	}
	if value.Scheme == "https" {
		return nil
	}
	if value.Scheme == "http" && isLoopbackHostname(value.Hostname()) {
		return nil
	}
	return fmt.Errorf("OAuth redirect URL must use HTTPS unless it targets localhost or a loopback IP address")
}

func isLoopbackHostname(hostname string) bool {
	return strings.EqualFold(hostname, "localhost") || net.ParseIP(hostname).IsLoopback()
}

func readBounded(body io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("OAuth response exceeds %d-byte limit", maxBytes)
	}
	return data, nil
}
