package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"kandev-plugin-bitbucket/internal/auth"

	"github.com/kdlbs/kandev/pluginsdk"
)

const credentialCleanupTimeout = 5 * time.Second

type oauthRegistrationSecret struct {
	ClientSecret string `json:"client_secret"`
}

type secretWrite struct {
	key   string
	value string
}

func (r *ConnectionResolver) oauthRegistration(ctx context.Context, workspaceID string, settings ConnectionSettings) (auth.OAuthRegistration, error) {
	secret, found, err := r.loadOAuthRegistrationSecret(ctx, workspaceID, settings.OAuthGeneration)
	if err != nil || !found {
		return auth.OAuthRegistration{}, fmt.Errorf("Bitbucket OAuth client registration is unavailable")
	}
	var saved oauthRegistrationSecret
	if err := json.Unmarshal([]byte(secret), &saved); err != nil || strings.TrimSpace(saved.ClientSecret) == "" {
		return auth.OAuthRegistration{}, fmt.Errorf("Bitbucket OAuth client registration is unavailable")
	}
	authorizationURL, tokenURL, endpointErr := r.oauthEndpoints(settings)
	redirectURL, redirectErr := url.Parse(settings.OAuthRedirectURL)
	if endpointErr != nil || redirectErr != nil {
		return auth.OAuthRegistration{}, fmt.Errorf("Bitbucket OAuth endpoints are invalid")
	}
	return auth.OAuthRegistration{
		ClientID: settings.OAuthClientID, ClientSecret: saved.ClientSecret,
		AuthorizationURL: authorizationURL, TokenURL: tokenURL, RedirectURL: redirectURL,
	}, nil
}

func (r *ConnectionResolver) connectionSecretWrites(
	ctx context.Context,
	workspaceID string,
	settings, previous ConnectionSettings,
	found bool,
	input ConnectionInput,
) ([]secretWrite, error) {
	writes := make([]secretWrite, 0, 2)
	if settings.AuthMethod == "oauth" && strings.TrimSpace(input.OAuthClientSecret) != "" {
		encoded, _ := json.Marshal(oauthRegistrationSecret{ClientSecret: input.OAuthClientSecret})
		writes = append(writes, secretWrite{
			key: oauthRegistrationSecretKey(workspaceID, settings.OAuthGeneration), value: string(encoded),
		})
	}
	if settings.AuthMethod != "oauth" {
		token := strings.TrimSpace(input.Token)
		if token == "" {
			if !found {
				return nil, invalidConnectionInput("Bitbucket token credential is required")
			}
			stored, present, err := r.loadTokenCredential(ctx, workspaceID, previous.CredentialGeneration)
			if err != nil {
				return nil, fmt.Errorf("load Bitbucket credential: %w", err)
			}
			if !present || strings.TrimSpace(stored) == "" {
				return nil, invalidConnectionInput("Bitbucket token credential is required")
			}
			token = stored
		}
		writes = append(writes, secretWrite{
			key: connectionSecretKey(workspaceID, settings.CredentialGeneration), value: token,
		})
	}
	return writes, nil
}

func (r *ConnectionResolver) writeSecrets(ctx context.Context, writes []secretWrite) error {
	for _, write := range writes {
		if err := r.host.SetSecret(ctx, write.key, write.value); err != nil {
			return fmt.Errorf("store Bitbucket credential: %w", err)
		}
	}
	return nil
}

func (r *ConnectionResolver) compensateStagedSecrets(ctx context.Context, writes []secretWrite) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), credentialCleanupTimeout)
	defer cancel()
	keys := make([]string, 0, len(writes))
	for _, write := range writes {
		keys = append(keys, write.key)
	}
	_ = r.deleteSecrets(cleanupCtx, uniqueSecretKeys(keys)...)
}

func (r *ConnectionResolver) cleanupCommittedSecrets(
	ctx context.Context,
	workspaceID string,
	settings ConnectionSettings,
) {
	if len(settings.PendingSecretRevocations) == 0 {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), credentialCleanupTimeout)
	defer cancel()
	if err := r.deleteSecrets(cleanupCtx, settings.PendingSecretRevocations...); err != nil {
		return
	}
	settings.PendingSecretRevocations = nil
	value, err := encodeState(settings)
	if err != nil {
		return
	}
	_ = r.host.SetState(cleanupCtx, "workspace", workspaceID, connectionStateKey, value)
}

func uniqueSecretKeys(keys []string) []string {
	result := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func supersededCredentialKeys(workspaceID string, previous ConnectionSettings, found bool, next ConnectionSettings) []string {
	if !found {
		return nil
	}
	previousOAuth := previous.AuthMethod == "oauth"
	nextOAuth := next.AuthMethod == "oauth"
	keys := make([]string, 0, 8)
	if previousOAuth && (!nextOAuth || previous.OAuthGeneration != next.OAuthGeneration) {
		keys = append(keys, oauthStateSecretKey(workspaceID))
		if previous.OAuthGeneration != 0 {
			keys = append(keys, oauthCredentialSecretKey(workspaceID, previous.OAuthGeneration))
			keys = append(keys, oauthRegistrationSecretKey(workspaceID, previous.OAuthGeneration))
		}
		keys = append(keys, legacyOAuthRegistrationSecretKey(workspaceID))
		if !nextOAuth {
			keys = append(keys, legacyOAuthRegistrationSecretKey(workspaceID))
		}
	}
	if !previousOAuth && (nextOAuth || previous.CredentialGeneration != next.CredentialGeneration) {
		keys = append(
			keys,
			connectionSecretKey(workspaceID, previous.CredentialGeneration),
			legacyConnectionSecretKey(workspaceID),
		)
	}
	return uniqueSecretKeys(keys)
}

func (r *ConnectionResolver) deleteSecrets(ctx context.Context, keys ...string) error {
	var firstErr error
	for _, key := range keys {
		if err := r.host.DeleteSecret(ctx, key); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("delete Bitbucket credential: %w", firstErr)
	}
	return nil
}

func (r *ConnectionResolver) loadTokenCredential(
	ctx context.Context,
	workspaceID string,
	generation uint64,
) (string, bool, error) {
	keys := make([]string, 0, 2)
	if generation != 0 {
		keys = append(keys, connectionSecretKey(workspaceID, generation))
	}
	keys = append(keys, legacyConnectionSecretKey(workspaceID))
	return r.firstSecret(ctx, keys)
}

func (r *ConnectionResolver) loadOAuthRegistrationSecret(
	ctx context.Context,
	workspaceID string,
	generation uint64,
) (string, bool, error) {
	keys := make([]string, 0, 2)
	if generation != 0 {
		keys = append(keys, oauthRegistrationSecretKey(workspaceID, generation))
	}
	keys = append(keys, legacyOAuthRegistrationSecretKey(workspaceID))
	return r.firstSecret(ctx, keys)
}

func (r *ConnectionResolver) firstSecret(ctx context.Context, keys []string) (string, bool, error) {
	for _, key := range keys {
		value, found, err := r.host.GetSecret(ctx, key)
		if err != nil {
			return "", false, err
		}
		if found {
			return value, true, nil
		}
	}
	return "", false, nil
}

func oauthRegistrationSecretKey(workspaceID string, generation uint64) string {
	return fmt.Sprintf("%s.%d", legacyOAuthRegistrationSecretKey(workspaceID), generation)
}

func legacyOAuthRegistrationSecretKey(workspaceID string) string {
	return workspaceSecretKey("bitbucket.oauth.registration.", workspaceID)
}

func oauthStateSecretKey(workspaceID string) string {
	return workspaceSecretKey("bitbucket.oauth.state.", workspaceID)
}

func oauthCredentialSecretKey(workspaceID string, generation uint64) string {
	return fmt.Sprintf("%s.%d", workspaceSecretKey("bitbucket.oauth.credential.", workspaceID), generation)
}

func workspaceSecretKey(prefix, workspaceID string) string {
	digest := sha256.Sum256([]byte(workspaceID))
	return prefix + hex.EncodeToString(digest[:16])
}

type hostOAuthCredentialRepository struct{ host pluginsdk.Host }

func (r hostOAuthCredentialRepository) LoadCredential(ctx context.Context, scope auth.CredentialScope) (auth.Credential, error) {
	encoded, found, err := r.host.GetSecret(ctx, oauthCredentialSecretKey(scope.WorkspaceID, scope.Generation))
	if err != nil || !found {
		return auth.Credential{}, fmt.Errorf("Bitbucket OAuth credential is unavailable")
	}
	var credential auth.Credential
	if err := json.Unmarshal([]byte(encoded), &credential); err != nil {
		return auth.Credential{}, fmt.Errorf("Bitbucket OAuth credential is unavailable")
	}
	return credential, nil
}

func (r hostOAuthCredentialRepository) SaveCredential(ctx context.Context, scope auth.CredentialScope, credential auth.Credential) error {
	encoded, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	if err := r.host.SetSecret(ctx, oauthCredentialSecretKey(scope.WorkspaceID, scope.Generation), string(encoded)); err != nil {
		return fmt.Errorf("store Bitbucket OAuth credential: %w", err)
	}
	return nil
}

func connectionSecretKey(workspaceID string, generation uint64) string {
	return fmt.Sprintf("%s.%d", legacyConnectionSecretKey(workspaceID), generation)
}

func legacyConnectionSecretKey(workspaceID string) string {
	digest := sha256.Sum256([]byte(workspaceID))
	return "bitbucket.token." + hex.EncodeToString(digest[:16])
}

type hostTokenSource struct {
	host pluginsdk.Host
	keys []string
}

func (s hostTokenSource) AccessToken(ctx context.Context) (string, error) {
	for _, key := range s.keys {
		token, found, err := s.host.GetSecret(ctx, key)
		if err != nil {
			return "", fmt.Errorf("Bitbucket authentication is required")
		}
		if found && strings.TrimSpace(token) != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("Bitbucket authentication is required")
}
