package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kandev/kandev/pkg/pluginsdk"
	"kandev-plugin-bitbucket/internal/cloud"
	"kandev-plugin-bitbucket/internal/domain"
)

func (r *Runtime) InvokeAgentTool(ctx context.Context, request *pluginsdk.AgentToolRequest) (*pluginsdk.AgentToolResult, error) {
	workflows, err := r.ready()
	if err != nil {
		return sourceToolError("bitbucket_source_unavailable"), nil
	}
	return workflows.InvokeSourceTool(ctx, request), nil
}
func sourceToolError(code string) *pluginsdk.AgentToolResult {
	return &pluginsdk.AgentToolResult{Text: code, IsError: true, StructuredContent: map[string]any{"error": code}}
}

func (w *Workflows) InvokeSourceTool(ctx context.Context, request *pluginsdk.AgentToolRequest) *pluginsdk.AgentToolResult {
	if request == nil || request.Context.TaskID == "" || request.Context.SessionID == "" || request.Context.WorkspaceID == "" || (request.Context.Surface != "kanban-task" && request.Context.Surface != "office-task") {
		return sourceToolError("bitbucket_source_scope_denied")
	}
	if request.Name != "source_read" && request.Name != "source_create" {
		return sourceToolError("bitbucket_source_operation_unsupported")
	}
	// The selector can choose only an entry already authorized by the host.
	arguments := make(map[string]any, len(request.Arguments))
	for key, value := range request.Arguments {
		arguments[key] = value
	}
	repositoryID, _ := arguments["repository_id"].(string)
	delete(arguments, "repository_id")
	var action pluginsdk.VerifiedActionContext
	matches := 0
	for _, candidate := range request.Context.SourceRepositories {
		if candidate.RepositoryID == repositoryID && candidate.WorkspaceID == request.Context.WorkspaceID && candidate.TaskID == request.Context.TaskID && candidate.SessionID == request.Context.SessionID && candidate.HeadBranch != "" {
			action = candidate
			matches++
		}
	}
	if matches != 1 {
		return sourceToolError("bitbucket_source_scope_denied")
	}
	body, err := json.Marshal(arguments)
	if err != nil {
		return sourceToolError("bitbucket_source_input_invalid")
	}
	key := "source.read"
	if request.Name == "source_create" {
		key = "source.create"
	}
	response, err := w.handleSourceAction(ctx, &pluginsdk.PluginActionRequest{ActionKey: key, Context: action, Body: body})
	if err != nil {
		return sourceToolError(err.Error())
	}
	var structured map[string]any
	if json.Unmarshal(response.Body, &structured) != nil {
		return sourceToolError("bitbucket_source_response_invalid")
	}
	return &pluginsdk.AgentToolResult{Text: string(response.Body), StructuredContent: structured}
}

func (w *Workflows) handleSourceAction(ctx context.Context, request *pluginsdk.PluginActionRequest) (*pluginsdk.PluginActionResponse, error) {
	var input cloud.SourceInput
	if len(request.Body) > 40000 || decodeAction(request.Body, &input) != nil {
		return nil, fmt.Errorf("bitbucket_source_input_invalid")
	}
	if request.ActionKey == "source.create" {
		if input.Operation != "" && input.Operation != "pr_create" {
			return nil, fmt.Errorf("bitbucket_source_operation_unsupported")
		}
		input.Operation = "pr_create"
	} else if input.Operation == "pr_create" {
		return nil, fmt.Errorf("bitbucket_source_operation_unsupported")
	}
	resolver, ok := w.resolver.(*ConnectionResolver)
	if !ok {
		return nil, fmt.Errorf("bitbucket_source_binding_unavailable")
	}
	result, err := resolver.sourceOperation(ctx, request.Context, input)
	if err != nil {
		return nil, err
	}
	return actionResponse(result)
}

func (r *ConnectionResolver) sourceOperation(ctx context.Context, action pluginsdk.VerifiedActionContext, input cloud.SourceInput) (map[string]any, error) {
	// Serialize create/reuse checks within this plugin process. External writers
	// remain subject to the provider's conflict response; writes are never retried.
	if !r.sourceMu.TryLock() {
		return nil, fmt.Errorf("bitbucket_source_busy")
	}
	defer r.sourceMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	repository, destination, err := r.sourceRepository(ctx, action)
	if err != nil {
		return nil, err
	}
	settings, found, err := r.loadConnectionRecord(ctx, action.WorkspaceID)
	if errors.Is(err, errRegisteredEmailRequired) {
		return nil, fmt.Errorf("bitbucket_registered_email_required")
	}
	if err != nil || !found || settings.Disconnected || settings.CredentialGeneration == 0 || settings.ConnectionBinding == "" {
		return nil, fmt.Errorf("bitbucket_source_binding_unavailable")
	}
	if settings.Product != domain.ProductCloud {
		return nil, fmt.Errorf("bitbucket_source_product_unsupported")
	}
	authentication, err := cloudAuthentication(settings)
	if err != nil {
		return nil, fmt.Errorf("bitbucket_api_auth_kind_unsupported")
	}
	// Source API operations never try a legacy secret key or another connection.
	var tokens cloud.TokenSource = hostTokenSource{host: r.host, keys: []string{connectionSecretKey(action.WorkspaceID, settings.CredentialGeneration)}}
	if settings.AuthMethod == "oauth" {
		tokens, err = r.tokenSource(ctx, action.WorkspaceID, settings)
		if err != nil {
			return nil, fmt.Errorf("bitbucket_api_credential_unavailable")
		}
	}
	guarded := sourceTokenSource{resolver: r, action: action, settings: settings, source: tokens, repository: repository, destination: destination}
	client := cloud.NewClient(cloud.ClientOptions{TokenSource: guarded, HTTPClient: r.httpClient, Authentication: authentication, APIBase: r.cloudAPIBaseOverride})
	result, err := client.SourceOperation(ctx, repository, branchName(action.HeadBranch), destination, input)
	if err != nil {
		return nil, err
	}
	if err = guarded.validate(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ConnectionResolver) sourceRepository(ctx context.Context, action pluginsdk.VerifiedActionContext) (domain.Repository, string, error) {
	denied := fmt.Errorf("bitbucket_source_scope_denied")
	if action.WorkspaceID == "" || action.TaskID == "" || action.SessionID == "" || action.RepositoryID == "" || action.HeadBranch == "" {
		return domain.Repository{}, "", denied
	}
	task, err := r.host.Tasks().Get(ctx, action.TaskID)
	if err != nil || task == nil || task.ID != action.TaskID || task.WorkspaceID != action.WorkspaceID {
		return domain.Repository{}, "", denied
	}
	var attachment *pluginsdk.TaskRepository
	for i := range task.Repositories {
		if task.Repositories[i].RepositoryID == action.RepositoryID {
			if attachment != nil {
				return domain.Repository{}, "", denied
			}
			attachment = &task.Repositories[i]
		}
	}
	if attachment == nil {
		return domain.Repository{}, "", denied
	}
	page := pluginsdk.Page{Limit: 100}
	for count := 0; count < 10; count++ {
		repositories, info, listErr := r.host.Repositories().List(ctx, action.WorkspaceID, page)
		if listErr != nil || len(repositories) > 100 {
			return domain.Repository{}, "", denied
		}
		for _, repository := range repositories {
			if repository.ID != action.RepositoryID {
				continue
			}
			if repository.WorkspaceID != action.WorkspaceID || repository.SourceType != "provider" || repository.ProviderID != "bitbucket" || repository.ProviderScope != "https://bitbucket.org" || repository.ProviderRepositoryID == "" {
				return domain.Repository{}, "", denied
			}
			clone, parseErr := url.Parse(repository.RemoteURL)
			if parseErr != nil || clone.Scheme != "https" || clone.Host != "bitbucket.org" || clone.User != nil || clone.RawQuery != "" || clone.Fragment != "" || clone.RawPath != "" || !providerHostMatches(repository.ProviderHost, clone) {
				return domain.Repository{}, "", denied
			}
			name := providerRepositoryName(repository)
			if clone.Path != "/"+repository.OwnerOrProject+"/"+name+".git" {
				return domain.Repository{}, "", denied
			}
			destination := branchName(attachment.BaseBranch)
			if destination == "" {
				destination = branchName(stringValue(repository.DefaultBranch))
			}
			if destination == "" || destination == branchName(action.HeadBranch) {
				return domain.Repository{}, "", denied
			}
			return domain.Repository{ID: repository.ProviderRepositoryID, ProviderScope: repository.ProviderScope, Namespace: repository.OwnerOrProject, Slug: name, CloneURL: clone}, destination, nil
		}
		if info == nil || !info.HasMore || info.NextCursor == "" || info.NextCursor == page.Cursor {
			break
		}
		page.Cursor = info.NextCursor
	}
	return domain.Repository{}, "", denied
}

type sourceTokenSource struct {
	repository  domain.Repository
	destination string
	resolver    *ConnectionResolver
	action      pluginsdk.VerifiedActionContext
	settings    ConnectionSettings
	source      cloud.TokenSource
}

func (s sourceTokenSource) validate(ctx context.Context) error {
	settings, found, err := s.resolver.loadConnectionRecord(ctx, s.action.WorkspaceID)
	if err != nil || !found || settings.Disconnected || settings.ConnectionBinding != s.settings.ConnectionBinding || settings.CredentialGeneration != s.settings.CredentialGeneration || settings.AuthMethod != s.settings.AuthMethod || settings.AuthIdentity != s.settings.AuthIdentity {
		return fmt.Errorf("bitbucket_source_binding_revoked")
	}
	repository, destination, err := s.resolver.sourceRepository(ctx, s.action)
	if err != nil {
		return err
	}
	if repository.ID != s.repository.ID || repository.ProviderScope != s.repository.ProviderScope || repository.Namespace != s.repository.Namespace || repository.Slug != s.repository.Slug || destination != s.destination {
		return fmt.Errorf("bitbucket_source_scope_denied")
	}
	return nil
}
func (s sourceTokenSource) AccessToken(ctx context.Context) (string, error) {
	token, _, err := s.AccessTokenCredential(ctx)
	return token, err
}

func (s sourceTokenSource) AccessTokenCredential(ctx context.Context) (string, time.Time, error) {
	if err := s.validate(ctx); err != nil {
		return "", time.Time{}, err
	}
	var token string
	var err error
	expiry := time.Now().Add(5 * time.Minute)
	if source, ok := s.source.(interface {
		AccessTokenCredential(context.Context) (string, time.Time, error)
	}); ok {
		token, expiry, err = source.AccessTokenCredential(ctx)
	} else {
		token, err = s.source.AccessToken(ctx)
	}
	if err != nil || strings.TrimSpace(token) == "" {
		return "", time.Time{}, fmt.Errorf("bitbucket_api_credential_unavailable")
	}
	if err = s.validate(ctx); err != nil {
		return "", time.Time{}, err
	}
	return token, expiry, nil
}

// sourceActionFailure preserves machine-readable source diagnostics while
// keeping unexpected errors behind the existing redacted action boundary.
func sourceActionFailure(err error) (*pluginsdk.PluginActionResponse, error) {
	statuses := map[string]int{
		"bitbucket_api_identity_unauthorized":    401,
		"bitbucket_api_permission_denied":        403,
		"bitbucket_rest_resource_unavailable":    404,
		"bitbucket_registered_email_required":    400,
		"bitbucket_source_input_invalid":         400,
		"bitbucket_source_scope_denied":          403,
		"bitbucket_source_binding_unavailable":   503,
		"bitbucket_source_binding_revoked":       403,
		"bitbucket_api_credential_unavailable":   401,
		"bitbucket_api_auth_kind_unsupported":    400,
		"bitbucket_source_operation_unsupported": 400,
		"bitbucket_source_product_unsupported":   400,
		"bitbucket_pr_lookup_incomplete":         409,
		"bitbucket_pr_source_ambiguous":          409,
		"bitbucket_source_busy":                  429,
	}
	code := err.Error()
	status, ok := statuses[code]
	if !ok {
		return actionFailureResponse(err)
	}
	body, _ := json.Marshal(map[string]string{"error": code, "code": code})
	return &pluginsdk.PluginActionResponse{Status: status, Body: body, Headers: map[string]string{"Content-Type": "application/json"}}, nil
}
