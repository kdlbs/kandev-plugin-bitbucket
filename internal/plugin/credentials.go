package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kdlbs/kandev/pluginsdk"
)

var ErrCredentialUnavailable = errors.New("Bitbucket Git credential unavailable")

type GitCredentialScope struct {
	WorkspaceID  string
	TaskID       string
	SessionID    string
	RepositoryID string
	Host         string
	Path         string
}

type GitCredential struct {
	Username  string
	Secret    string
	ExpiresAt time.Time
}

// GitCredentialSource resolves transient credentials for one exact
// host-verified scope. It must not persist the returned Secret.
type GitCredentialSource interface {
	ResolveGitCredential(context.Context, GitCredentialScope) (GitCredential, error)
	GetGitCredentialBinding(context.Context, GitCredentialScope) (string, error)
}

type CredentialResolver struct{ source GitCredentialSource }

func NewCredentialResolver(source GitCredentialSource) (*CredentialResolver, error) {
	if source == nil {
		return nil, fmt.Errorf("Git credential source is required")
	}
	return &CredentialResolver{source: source}, nil
}

func (r *CredentialResolver) ResolveGitCredential(ctx context.Context, request *pluginsdk.ResolveGitCredentialRequest) (*pluginsdk.ResolveGitCredentialResponse, error) {
	if request == nil || !strings.EqualFold(request.ProviderID, "bitbucket") || !validCredentialScope(request) {
		return nil, ErrCredentialUnavailable
	}
	credential, err := r.source.ResolveGitCredential(ctx, GitCredentialScope{
		WorkspaceID: request.WorkspaceID, TaskID: request.TaskID, SessionID: request.SessionID,
		RepositoryID: request.RepositoryID, Host: request.Host, Path: request.Path,
	})
	if err != nil || credential.Username == "" || credential.Secret == "" || credential.ExpiresAt.IsZero() || !credential.ExpiresAt.After(time.Now()) {
		return nil, ErrCredentialUnavailable
	}
	return &pluginsdk.ResolveGitCredentialResponse{
		Username:  credential.Username,
		Secret:    credential.Secret,
		ExpiresAt: credential.ExpiresAt.UTC().Format(time.RFC3339),
	}, nil
}

// GetGitCredentialBinding returns a non-secret opaque connection revision for
// the exact host-verified lease scope. It deliberately never resolves a Git
// credential, so a host can invalidate stale leases without reading a token.
func (r *CredentialResolver) GetGitCredentialBinding(ctx context.Context, request *pluginsdk.GitCredentialBindingRequest) (*pluginsdk.GitCredentialBindingResponse, error) {
	if request == nil || !strings.EqualFold(request.ProviderID, "bitbucket") || !validCredentialBindingScope(request) {
		return nil, ErrCredentialUnavailable
	}
	binding, err := r.source.GetGitCredentialBinding(ctx, credentialBindingScope(request))
	if err != nil || strings.TrimSpace(binding) == "" {
		return nil, ErrCredentialUnavailable
	}
	return &pluginsdk.GitCredentialBindingResponse{Binding: binding}, nil
}

func validCredentialScope(request *pluginsdk.ResolveGitCredentialRequest) bool {
	return request.WorkspaceID != "" && request.TaskID != "" && request.SessionID != "" && request.RepositoryID != "" && request.Host != "" && strings.HasPrefix(request.Path, "/")
}

func validCredentialBindingScope(request *pluginsdk.GitCredentialBindingRequest) bool {
	return request.WorkspaceID != "" && request.TaskID != "" && request.SessionID != "" && request.RepositoryID != "" && request.Host != "" && strings.HasPrefix(request.Path, "/")
}

func credentialBindingScope(request *pluginsdk.GitCredentialBindingRequest) GitCredentialScope {
	return GitCredentialScope{WorkspaceID: request.WorkspaceID, TaskID: request.TaskID, SessionID: request.SessionID, RepositoryID: request.RepositoryID, Host: request.Host, Path: request.Path}
}
