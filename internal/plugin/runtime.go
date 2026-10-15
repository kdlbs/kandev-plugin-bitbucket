package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/kandev/kandev/pkg/pluginsdk"
)

// Runtime is the server-facing plugin implementation. Host injection happens
// once per subprocess; handlers return a safe readiness error until then.
type Runtime struct {
	pluginsdk.UnimplementedPlugin

	mu        sync.RWMutex
	workflows *Workflows
	err       error
	cancel    context.CancelFunc
}

func NewRuntime() *Runtime { return &Runtime{} }

func (r *Runtime) SetHost(host pluginsdk.Host) {
	r.UnimplementedPlugin.SetHost(host)
	resolver, err := NewConnectionResolver(host)
	if err != nil {
		r.setUnavailable(err)
		return
	}
	workflows, err := NewWorkflows(host, resolver)
	if err != nil {
		r.setUnavailable(err)
		return
	}
	manager, err := NewManager(host, resolver, workflows.watches)
	if err != nil {
		r.setUnavailable(err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	previousCancel := r.cancel
	r.workflows = workflows
	r.err = nil
	r.cancel = cancel
	r.mu.Unlock()
	if previousCancel != nil {
		previousCancel()
	}
	go manager.Run(ctx)
}

func (r *Runtime) setUnavailable(err error) {
	r.mu.Lock()
	previousCancel := r.cancel
	r.workflows = nil
	r.err = err
	r.cancel = nil
	r.mu.Unlock()
	if previousCancel != nil {
		previousCancel()
	}
}

func (r *Runtime) HandleAction(ctx context.Context, request *pluginsdk.PluginActionRequest) (*pluginsdk.PluginActionResponse, error) {
	workflows, err := r.ready()
	if err != nil {
		return nil, err
	}
	response, err := workflows.HandleAction(ctx, request)
	if err != nil {
		if request != nil && (request.ActionKey == "source.read" || request.ActionKey == "source.create") {
			return sourceActionFailure(err)
		}
		return actionFailureResponse(err)
	}
	return response, nil
}

func (r *Runtime) HandleWebhook(ctx context.Context, request *pluginsdk.WebhookRequest) (*pluginsdk.WebhookResponse, error) {
	workflows, err := r.ready()
	if err != nil {
		return &pluginsdk.WebhookResponse{Status: 503}, nil
	}
	return workflows.HandleWebhook(ctx, request)
}

func (r *Runtime) SearchEntityReferences(ctx context.Context, request *pluginsdk.SearchEntityReferencesRequest) (*pluginsdk.SearchEntityReferencesResponse, error) {
	workflows, err := r.ready()
	if err != nil {
		return nil, err
	}
	return workflows.SearchEntityReferences(ctx, request)
}

func (r *Runtime) AuthorizeEntityReference(ctx context.Context, request *pluginsdk.AuthorizeEntityReferenceRequest) (*pluginsdk.AuthorizeEntityReferenceResponse, error) {
	workflows, err := r.ready()
	if err != nil {
		return nil, err
	}
	return workflows.AuthorizeEntityReference(ctx, request)
}

func (r *Runtime) ResolveGitCredential(ctx context.Context, request *pluginsdk.ResolveGitCredentialRequest) (*pluginsdk.ResolveGitCredentialResponse, error) {
	workflows, err := r.ready()
	if err != nil {
		return nil, ErrCredentialUnavailable
	}
	return workflows.ResolveGitCredential(ctx, request)
}

func (r *Runtime) GetGitCredentialBinding(ctx context.Context, request *pluginsdk.GitCredentialBindingRequest) (*pluginsdk.GitCredentialBindingResponse, error) {
	workflows, err := r.ready()
	if err != nil {
		return nil, ErrCredentialUnavailable
	}
	return workflows.GetGitCredentialBinding(ctx, request)
}

func (r *Runtime) ready() (*Workflows, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.err != nil {
		return nil, fmt.Errorf("Bitbucket plugin is unavailable")
	}
	if r.workflows == nil {
		return nil, fmt.Errorf("Bitbucket plugin is not ready")
	}
	return r.workflows, nil
}

var (
	_ pluginsdk.Plugin                 = (*Runtime)(nil)
	_ pluginsdk.ActionHandler          = (*Runtime)(nil)
	_ pluginsdk.EntityReferenceHandler = (*Runtime)(nil)
	_ pluginsdk.GitCredentialHandler   = (*Runtime)(nil)
	_ pluginsdk.HostSetter             = (*Runtime)(nil)
)
