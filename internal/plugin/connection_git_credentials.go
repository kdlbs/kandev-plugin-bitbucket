package plugin

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"kandev-plugin-bitbucket/internal/domain"

	"github.com/kdlbs/kandev/pluginsdk"
)

// ValidateGitCredentialScope proves that a host-verified broker request still
// targets this workspace's configured Bitbucket clone endpoint. The adapter
// token is never returned until this check succeeds.
func (r *ConnectionResolver) ValidateGitCredentialScope(ctx context.Context, scope GitCredentialScope) error {
	settings, found, err := r.Load(ctx, scope.WorkspaceID)
	if err != nil || !found || scope.TaskID == "" || scope.RepositoryID == "" || scope.Host == "" ||
		!strings.HasPrefix(scope.Path, "/") || strings.ContainsAny(scope.Path, "?#\\") {
		return ErrCredentialUnavailable
	}
	hostURL, err := url.Parse("https://" + scope.Host)
	if err != nil || hostURL.Host != scope.Host || hostURL.Path != "" || hostURL.RawQuery != "" || hostURL.Fragment != "" {
		return ErrCredentialUnavailable
	}
	if r.matchesTaskRepositoryScope(ctx, settings, scope) {
		return nil
	}
	return ErrCredentialUnavailable
}

// GitCredentialBinding returns the current non-secret connection generation
// only after the exact task/repository clone scope remains valid. It never
// constructs a provider client or reads a credential secret.
func (r *ConnectionResolver) GitCredentialBinding(ctx context.Context, scope GitCredentialScope) (string, error) {
	settings, found, err := r.Load(ctx, scope.WorkspaceID)
	if err != nil || !found || settings.CredentialGeneration == 0 {
		return "", ErrCredentialUnavailable
	}
	if err := r.ValidateGitCredentialScope(ctx, scope); err != nil {
		return "", ErrCredentialUnavailable
	}
	return fmt.Sprintf("bitbucket-credential:%d", settings.CredentialGeneration), nil
}

// matchesTaskRepositoryScope permits a fork only when the host-verified task
// points at one exact Bitbucket repository and its credential-free clone URL
// is the requested host/path. It never trusts a browser repository value.
func (r *ConnectionResolver) matchesTaskRepositoryScope(ctx context.Context, settings ConnectionSettings, scope GitCredentialScope) bool {
	task, err := r.host.Tasks().Get(ctx, scope.TaskID)
	if err != nil || task == nil || task.ID != scope.TaskID || task.WorkspaceID != scope.WorkspaceID {
		return false
	}
	matched := false
	for _, taskRepository := range task.Repositories {
		if taskRepository.RepositoryID == scope.RepositoryID {
			if matched {
				return false
			}
			matched = true
		}
	}
	if !matched {
		return false
	}
	page := pluginsdk.Page{Limit: 100}
	for {
		repositories, info, listErr := r.host.Repositories().List(ctx, scope.WorkspaceID, page)
		if listErr != nil {
			return false
		}
		for _, repository := range repositories {
			if repository.ID != scope.RepositoryID || repository.SourceType != "provider" || repository.ProviderID != "bitbucket" ||
				repository.ProviderRepositoryID == "" || repository.ProviderHost == "" || repository.OwnerOrProject == "" ||
				repository.ProviderName == "" || repository.RemoteURL == "" {
				continue
			}
			cloneURL, parseErr := url.Parse(repository.RemoteURL)
			if parseErr != nil || cloneURL.Scheme != "https" || cloneURL.User != nil || cloneURL.RawPath != "" ||
				cloneURL.RawQuery != "" || cloneURL.Fragment != "" || !strings.EqualFold(cloneURL.Host, scope.Host) ||
				!providerHostMatches(repository.ProviderHost, cloneURL) {
				return false
			}
			owner, name, valid := repositoryIdentity(settings, cloneURL)
			if !valid || !strings.EqualFold(repository.OwnerOrProject, owner) || repository.ProviderName != name {
				return false
			}
			return normalizedClonePath(cloneURL.Path) == normalizedClonePath(scope.Path)
		}
		if info == nil || !info.HasMore || info.NextCursor == "" {
			return false
		}
		page.Cursor = info.NextCursor
	}
}

// providerHostMatches accepts the current host contract (an HTTPS origin)
// plus the pre-origin legacy hostname while rejecting paths and credentials.
func providerHostMatches(raw string, cloneURL *url.URL) bool {
	value := strings.TrimSpace(raw)
	if value == "" || cloneURL == nil {
		return false
	}
	if !strings.Contains(value, "://") {
		return strings.EqualFold(value, cloneURL.Host)
	}
	origin, err := url.Parse(value)
	if err != nil || origin.Scheme != "https" || origin.User != nil || origin.Host == "" ||
		(origin.Path != "" && origin.Path != "/") || origin.RawPath != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	return strings.EqualFold(origin.Host, cloneURL.Host)
}

func repositoryIdentity(settings ConnectionSettings, cloneURL *url.URL) (string, string, bool) {
	if cloneURL == nil {
		return "", "", false
	}
	var relative string
	switch settings.Product {
	case domain.ProductCloud:
		if !strings.EqualFold(cloneURL.Host, "bitbucket.org") {
			return "", "", false
		}
		relative = strings.Trim(cloneURL.Path, "/")
	case domain.ProductDataCenter:
		base, err := url.Parse(settings.BaseURL)
		if err != nil || !strings.EqualFold(base.Host, cloneURL.Host) {
			return "", "", false
		}
		prefix := strings.TrimSuffix(base.Path, "/") + "/scm/"
		if !strings.HasPrefix(cloneURL.Path, prefix) {
			return "", "", false
		}
		relative = strings.TrimPrefix(cloneURL.Path, prefix)
	default:
		return "", "", false
	}
	parts := strings.Split(strings.Trim(relative, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	name := strings.TrimSuffix(parts[1], ".git")
	if name == "" {
		return "", "", false
	}
	return parts[0], name, true
}

func normalizedClonePath(value string) string {
	return strings.TrimSuffix(strings.TrimSuffix(value, "/"), ".git")
}
