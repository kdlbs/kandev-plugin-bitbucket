package plugin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kdlbs/kandev/pluginsdk"
	"github.com/stretchr/testify/require"
)

func TestCredentialResolver_ForwardsExactHostVerifiedScope(t *testing.T) {
	source := &recordingCredentialSource{credential: GitCredential{Username: "x-token-auth", Secret: "secret-value", ExpiresAt: time.Now().Add(time.Hour)}}
	resolver, err := NewCredentialResolver(source)
	require.NoError(t, err)

	response, err := resolver.ResolveGitCredential(context.Background(), &pluginsdk.ResolveGitCredentialRequest{
		ProviderID: "bitbucket", WorkspaceID: "workspace", TaskID: "task", SessionID: "session", RepositoryID: "repository", Host: "bitbucket.example.test", Path: "/scm/PRJ/repo.git",
	})
	require.NoError(t, err)
	require.Equal(t, "x-token-auth", response.Username)
	require.Equal(t, "secret-value", response.Secret)
	require.Equal(t, GitCredentialScope{WorkspaceID: "workspace", TaskID: "task", SessionID: "session", RepositoryID: "repository", Host: "bitbucket.example.test", Path: "/scm/PRJ/repo.git"}, source.scope)
}

func TestCredentialResolver_ReturnsNonSecretLiveBindingWithoutResolvingCredential(t *testing.T) {
	source := &recordingCredentialSource{binding: "bitbucket:workspace:7"}
	resolver, err := NewCredentialResolver(source)
	require.NoError(t, err)

	response, err := resolver.GetGitCredentialBinding(context.Background(), &pluginsdk.GitCredentialBindingRequest{
		ProviderID: "bitbucket", WorkspaceID: "workspace", TaskID: "task", SessionID: "session", RepositoryID: "repository", Host: "bitbucket.example.test", Path: "/scm/PRJ/repo.git",
	})

	require.NoError(t, err)
	require.Equal(t, "bitbucket:workspace:7", response.Binding)
	require.Zero(t, source.resolveCalls, "binding lookup must not resolve a credential secret")
}

func TestCredentialResolver_NeverLeaksSecretFromSourceError(t *testing.T) {
	resolver, err := NewCredentialResolver(&recordingCredentialSource{err: errors.New("remote rejected secret-value")})
	require.NoError(t, err)
	_, err = resolver.ResolveGitCredential(context.Background(), &pluginsdk.ResolveGitCredentialRequest{
		ProviderID: "bitbucket", WorkspaceID: "workspace", TaskID: "task", SessionID: "session", RepositoryID: "repository", Host: "bitbucket.org", Path: "/workspace/repo.git",
	})
	require.ErrorIs(t, err, ErrCredentialUnavailable)
	require.NotContains(t, strings.ToLower(err.Error()), "secret-value")
}

func TestCredentialResolver_RejectsIncompleteOrWrongProviderScope(t *testing.T) {
	resolver, err := NewCredentialResolver(&recordingCredentialSource{})
	require.NoError(t, err)
	_, err = resolver.ResolveGitCredential(context.Background(), &pluginsdk.ResolveGitCredentialRequest{ProviderID: "github"})
	require.ErrorIs(t, err, ErrCredentialUnavailable)
}

type recordingCredentialSource struct {
	scope        GitCredentialScope
	credential   GitCredential
	err          error
	binding      string
	resolveCalls int
}

func (s *recordingCredentialSource) ResolveGitCredential(_ context.Context, scope GitCredentialScope) (GitCredential, error) {
	s.scope = scope
	s.resolveCalls++
	return s.credential, s.err
}

func (s *recordingCredentialSource) GetGitCredentialBinding(_ context.Context, scope GitCredentialScope) (string, error) {
	s.scope = scope
	return s.binding, s.err
}
