package plugin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"kandev-plugin-bitbucket/internal/auth"
	"kandev-plugin-bitbucket/internal/cloud"
	"kandev-plugin-bitbucket/internal/datacenter"
	"kandev-plugin-bitbucket/internal/domain"

	"github.com/kandev/kandev/pkg/pluginsdk"
)

const connectionStateKey = "bitbucket.connection.v1"

var ErrInvalidConnectionInput = errors.New("invalid Bitbucket connection input")

func invalidConnectionInput(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidConnectionInput, fmt.Sprintf(format, args...))
}

// ConnectionInput accepts a credential only for the duration of the action.
// Token is deliberately excluded from ConnectionSettings and host state.
type ConnectionInput struct {
	Product           domain.Product `json:"product"`
	BaseURL           string         `json:"base_url"`
	CloudWorkspace    string         `json:"cloud_workspace"`
	AuthMethod        string         `json:"auth_method"`
	AuthIdentity      string         `json:"auth_identity"`
	Token             string         `json:"token"`
	OAuthClientID     string         `json:"oauth_client_id"`
	OAuthClientSecret string         `json:"oauth_client_secret"`
	OAuthRedirectURL  string         `json:"oauth_redirect_url"`
}

// ConnectionSettings is the credential-free workspace connection record.
type ConnectionSettings struct {
	Product              domain.Product `json:"product"`
	BaseURL              string         `json:"base_url,omitempty"`
	CloudWorkspace       string         `json:"cloud_workspace,omitempty"`
	AuthMethod           string         `json:"auth_method,omitempty"`
	AuthIdentity         string         `json:"auth_identity,omitempty"`
	OAuthClientID        string         `json:"oauth_client_id,omitempty"`
	OAuthRedirectURL     string         `json:"oauth_redirect_url,omitempty"`
	OAuthGeneration      uint64         `json:"oauth_generation,omitempty"`
	CredentialGeneration uint64         `json:"credential_generation,omitempty"`
	// ConnectionBinding is a non-secret epoch used to fence persisted watches
	// from a later account, tenant, or server connection.
	ConnectionBinding string `json:"connection_binding,omitempty"`
	// Secret identifiers pending best-effort deletion after a committed
	// generation switch. Keeping this with the credential-free pointer makes
	// revocation retryable across process restarts without exposing plaintext.
	PendingSecretRevocations []string `json:"pending_secret_revocations,omitempty"`
	// Disconnected marks a fail-closed tombstone while secret revocation is
	// pending. Load hides tombstones from providers, while Disconnect and Save
	// retain enough generation metadata to finish cleanup after a restart.
	Disconnected bool `json:"disconnected,omitempty"`
}

// ConnectionSettingsStore is the optional mutable part of ProviderResolver.
// Test resolvers can stay read-only while production resolver owns settings.
type ConnectionSettingsStore interface {
	Load(context.Context, string) (ConnectionSettings, bool, error)
	Save(context.Context, string, ConnectionInput) (ConnectionSettings, error)
}

// ConnectionResolver builds short-lived provider clients from host-owned
// connection state and secrets. It never caches or persists plaintext tokens.
type ConnectionResolver struct {
	sourceMu   sync.Mutex
	host       pluginsdk.Host
	httpClient *http.Client

	// Test-only seams. Production action input cannot override provider endpoints.
	cloudAPIBaseOverride  *url.URL
	oauthEndpointOverride *oauthEndpointPair

	refresherMu  sync.Mutex
	refresher    *auth.Refresher
	connectionMu sync.Mutex
	oauthMu      sync.Mutex
	oauthState   map[string]*auth.StateManager
	oauthFlows   map[string]oauthFlow
	oauthEpoch   map[string]uint64
}

func NewConnectionResolver(host pluginsdk.Host) (*ConnectionResolver, error) {
	if host == nil {
		return nil, fmt.Errorf("connection host is required")
	}
	return &ConnectionResolver{
		host: host, oauthState: make(map[string]*auth.StateManager), oauthFlows: make(map[string]oauthFlow), oauthEpoch: make(map[string]uint64),
	}, nil
}

func (r *ConnectionResolver) Load(ctx context.Context, workspaceID string) (ConnectionSettings, bool, error) {
	settings, found, err := r.loadConnectionRecord(ctx, workspaceID)
	if err != nil || !found {
		return settings, found, err
	}
	if settings.Disconnected {
		return ConnectionSettings{}, false, nil
	}
	if settings.ConnectionBinding == "" {
		settings, err = r.migrateConnectionBinding(ctx, workspaceID)
		if err != nil {
			return ConnectionSettings{}, false, err
		}
	}
	return settings, true, nil
}

func (r *ConnectionResolver) migrateConnectionBinding(ctx context.Context, workspaceID string) (ConnectionSettings, error) {
	r.connectionMu.Lock()
	defer r.connectionMu.Unlock()
	settings, found, err := r.loadConnectionRecord(ctx, workspaceID)
	if err != nil {
		return ConnectionSettings{}, err
	}
	if !found || settings.Disconnected {
		return ConnectionSettings{}, fmt.Errorf("Bitbucket connection is not configured")
	}
	if settings.ConnectionBinding != "" {
		return settings, nil
	}
	settings.ConnectionBinding, err = newConnectionBinding()
	if err != nil {
		return ConnectionSettings{}, err
	}
	value, err := encodeState(settings)
	if err != nil {
		return ConnectionSettings{}, fmt.Errorf("encode Bitbucket connection migration: %w", err)
	}
	if err := r.host.SetState(ctx, "workspace", workspaceID, connectionStateKey, value); err != nil {
		return ConnectionSettings{}, fmt.Errorf("save Bitbucket connection migration: %w", err)
	}
	return settings, nil
}

func (r *ConnectionResolver) loadConnectionRecord(ctx context.Context, workspaceID string) (ConnectionSettings, bool, error) {
	if workspaceID == "" {
		return ConnectionSettings{}, false, fmt.Errorf("workspace id is required")
	}
	value, found, err := r.host.GetState(ctx, "workspace", workspaceID, connectionStateKey)
	if err != nil {
		return ConnectionSettings{}, false, fmt.Errorf("load Bitbucket connection: %w", err)
	}
	if !found {
		return ConnectionSettings{}, false, nil
	}
	var settings ConnectionSettings
	if err := decodeState(value, &settings); err != nil {
		return ConnectionSettings{}, false, fmt.Errorf("decode Bitbucket connection: %w", err)
	}
	if settings.Disconnected {
		return settings, true, nil
	}
	if err := validateConnection(settings); err != nil {
		return ConnectionSettings{}, false, fmt.Errorf("stored Bitbucket connection is invalid: %w", err)
	}
	return settings, true, nil
}

func (r *ConnectionResolver) Save(ctx context.Context, workspaceID string, input ConnectionInput) (ConnectionSettings, error) {
	if workspaceID == "" {
		return ConnectionSettings{}, fmt.Errorf("workspace id is required")
	}
	r.connectionMu.Lock()
	defer r.connectionMu.Unlock()
	previous, found, err := r.loadConnectionRecord(ctx, workspaceID)
	if err != nil {
		return ConnectionSettings{}, err
	}
	if found && previous.Disconnected {
		if err := r.finishDisconnect(ctx, workspaceID, previous); err != nil {
			return ConnectionSettings{}, err
		}
		previous, found = ConnectionSettings{}, false
	}
	credentialGeneration, err := nextCredentialGeneration(previous, found)
	if err != nil {
		return ConnectionSettings{}, err
	}
	settings := ConnectionSettings{
		Product: input.Product, BaseURL: strings.TrimSpace(input.BaseURL),
		CloudWorkspace: strings.TrimSpace(input.CloudWorkspace), AuthMethod: normalizedAuthMethod(input.AuthMethod), AuthIdentity: strings.TrimSpace(input.AuthIdentity),
		CredentialGeneration: credentialGeneration,
	}
	if found && sameConnectionTarget(previous, settings) && previous.ConnectionBinding != "" {
		settings.ConnectionBinding = previous.ConnectionBinding
	} else {
		settings.ConnectionBinding, err = newConnectionBinding()
		if err != nil {
			return ConnectionSettings{}, err
		}
	}
	if settings.AuthMethod == "oauth" {
		if strings.TrimSpace(input.Token) != "" {
			return ConnectionSettings{}, invalidConnectionInput("Bitbucket OAuth does not accept a token credential")
		}
		if err := r.configureOAuthSettings(ctx, workspaceID, &settings, previous, found, input); err != nil {
			return ConnectionSettings{}, err
		}
	}
	if settings.CloudWorkspace == "" && settings.Product == domain.ProductCloud {
		return ConnectionSettings{}, invalidConnectionInput("Bitbucket Cloud workspace is required")
	}
	if err := validateConnection(settings); err != nil {
		return ConnectionSettings{}, invalidConnectionInput("%v", err)
	}
	if err := r.validateTokenCredential(ctx, workspaceID, settings, previous, found, input.Token); err != nil {
		return ConnectionSettings{}, err
	}
	writes, err := r.connectionSecretWrites(ctx, workspaceID, settings, previous, found, input)
	if err != nil {
		return ConnectionSettings{}, err
	}
	deleteKeys := supersededCredentialKeys(workspaceID, previous, found, settings)
	settings.PendingSecretRevocations = uniqueSecretKeys(append(previous.PendingSecretRevocations, deleteKeys...))
	if err := r.writeSecrets(ctx, writes); err != nil {
		r.compensateStagedSecrets(ctx, writes)
		return ConnectionSettings{}, err
	}
	value, err := encodeState(settings)
	if err != nil {
		r.compensateStagedSecrets(ctx, writes)
		return ConnectionSettings{}, fmt.Errorf("encode Bitbucket connection: %w", err)
	}
	if err := r.host.SetState(ctx, "workspace", workspaceID, connectionStateKey, value); err != nil {
		r.compensateStagedSecrets(ctx, writes)
		return ConnectionSettings{}, fmt.Errorf("save Bitbucket connection: %w", err)
	}
	r.invalidateSupersededOAuth(workspaceID, previous, found, settings)
	r.cleanupCommittedSecrets(ctx, workspaceID, settings)
	settings.PendingSecretRevocations = nil
	return settings, nil
}

func (r *ConnectionResolver) validateTokenCredential(
	ctx context.Context,
	workspaceID string,
	settings, previous ConnectionSettings,
	found bool,
	token string,
) error {
	if settings.AuthMethod == "oauth" || strings.TrimSpace(token) != "" {
		return nil
	}
	if !found || previous.Product != settings.Product || previous.AuthMethod != settings.AuthMethod || previous.AuthIdentity != settings.AuthIdentity {
		return invalidConnectionInput("Bitbucket token credential is required")
	}
	stored, present, err := r.loadTokenCredential(ctx, workspaceID, previous.CredentialGeneration)
	if err != nil {
		return fmt.Errorf("load Bitbucket credential: %w", err)
	}
	if !present || strings.TrimSpace(stored) == "" {
		return invalidConnectionInput("Bitbucket token credential is required")
	}
	return nil
}

// Disconnect removes every workspace-scoped connection artifact. It first
// commits a fail-closed tombstone so a secret deletion failure remains
// retryable without allowing fresh provider or broker resolution.
func (r *ConnectionResolver) Disconnect(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}
	r.connectionMu.Lock()
	defer r.connectionMu.Unlock()
	settings, found, err := r.loadConnectionRecord(ctx, workspaceID)
	if err != nil {
		return err
	}
	r.invalidateOAuthWorkspace(workspaceID, settings, found)
	if found && settings.Disconnected {
		return r.finishDisconnect(ctx, workspaceID, settings)
	}
	keys := []string{
		legacyConnectionSecretKey(workspaceID), legacyOAuthRegistrationSecretKey(workspaceID),
		oauthStateSecretKey(workspaceID),
	}
	if found {
		keys = append(keys, settings.PendingSecretRevocations...)
		keys = append(keys,
			connectionSecretKey(workspaceID, settings.CredentialGeneration),
			oauthRegistrationSecretKey(workspaceID, settings.OAuthGeneration),
		)
		if settings.OAuthGeneration != 0 {
			keys = append(keys, oauthCredentialSecretKey(workspaceID, settings.OAuthGeneration))
		}
	}
	tombstone := ConnectionSettings{Disconnected: true, PendingSecretRevocations: uniqueSecretKeys(keys)}
	value, err := encodeState(tombstone)
	if err != nil {
		return fmt.Errorf("encode Bitbucket disconnect state: %w", err)
	}
	if err := r.host.SetState(ctx, "workspace", workspaceID, connectionStateKey, value); err != nil {
		return fmt.Errorf("save Bitbucket disconnect state: %w", err)
	}
	return r.finishDisconnect(ctx, workspaceID, tombstone)
}

func (r *ConnectionResolver) finishDisconnect(ctx context.Context, workspaceID string, tombstone ConnectionSettings) error {
	if err := r.deleteSecrets(ctx, tombstone.PendingSecretRevocations...); err != nil {
		return err
	}
	if err := r.host.DeleteState(ctx, "workspace", workspaceID, connectionStateKey); err != nil {
		return fmt.Errorf("delete Bitbucket connection: %w", err)
	}
	return nil
}

func nextCredentialGeneration(previous ConnectionSettings, found bool) (uint64, error) {
	if !found || previous.CredentialGeneration == 0 {
		return 1, nil
	}
	if previous.CredentialGeneration == ^uint64(0) {
		return 0, fmt.Errorf("Bitbucket credential generation exhausted")
	}
	return previous.CredentialGeneration + 1, nil
}

func sameConnectionTarget(previous, next ConnectionSettings) bool {
	return previous.Product == next.Product &&
		strings.TrimSpace(previous.BaseURL) == strings.TrimSpace(next.BaseURL) &&
		strings.EqualFold(strings.TrimSpace(previous.CloudWorkspace), strings.TrimSpace(next.CloudWorkspace)) &&
		previous.AuthMethod == next.AuthMethod &&
		strings.EqualFold(strings.TrimSpace(previous.AuthIdentity), strings.TrimSpace(next.AuthIdentity))
}

func newConnectionBinding() (string, error) {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("create Bitbucket connection binding: %w", err)
	}
	return "bitbucket-connection:v1:" + base64.RawURLEncoding.EncodeToString(value), nil
}

func (r *ConnectionResolver) Provider(ctx context.Context, workspaceID string) (domain.Provider, error) {
	settings, found, err := r.Load(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("Bitbucket connection is not configured")
	}
	tokens, err := r.tokenSource(ctx, workspaceID, settings)
	if err != nil {
		return nil, err
	}
	switch settings.Product {
	case domain.ProductCloud:
		authentication, authErr := cloudAuthentication(settings)
		if authErr != nil {
			return nil, authErr
		}
		options := cloud.ClientOptions{TokenSource: tokens, HTTPClient: r.httpClient, Authentication: authentication, APIBase: r.cloudAPIBaseOverride}
		return repositorySearchProvider{Provider: cloud.NewClient(options), workspace: settings.CloudWorkspace}, nil
	case domain.ProductDataCenter:
		authentication, authErr := dataCenterAuthentication(settings)
		if authErr != nil {
			return nil, authErr
		}
		provider, clientErr := datacenter.NewClient(datacenter.ClientOptions{
			ConnectionOptions: datacenter.ConnectionOptions{BaseURL: settings.BaseURL}, HTTPClient: r.httpClient, TokenSource: tokens, Authentication: authentication,
		})
		if clientErr != nil {
			return nil, clientErr
		}
		return repositorySearchProvider{Provider: provider}, nil
	default:
		return nil, fmt.Errorf("unsupported Bitbucket product")
	}
}

func (r *ConnectionResolver) tokenSource(ctx context.Context, workspaceID string, settings ConnectionSettings) (interface {
	AccessToken(context.Context) (string, error)
}, error) {
	if settings.AuthMethod != "oauth" {
		return hostTokenSource{
			host: r.host,
			keys: []string{
				connectionSecretKey(workspaceID, settings.CredentialGeneration),
				legacyConnectionSecretKey(workspaceID),
			},
		}, nil
	}
	registration, err := r.oauthRegistration(ctx, workspaceID, settings)
	if err != nil {
		return nil, err
	}
	scope := oauthCredentialScope(workspaceID, settings)
	return auth.NewRefreshingTokenSource(scope, registration, hostOAuthCredentialRepository{host: r.host}, r.oauthRefresher(), time.Now), nil
}

func (r *ConnectionResolver) oauthRefresher() *auth.Refresher {
	r.refresherMu.Lock()
	defer r.refresherMu.Unlock()
	if r.refresher == nil {
		r.refresher = auth.NewRefresher(r.httpClient, time.Now)
	}
	return r.refresher
}
