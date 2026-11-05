package plugin

import (
	"context"
	"errors"
	"net/url"

	"github.com/kdlbs/kandev/pluginsdk"
)

func (w *Workflows) handleConnectionAction(ctx context.Context, request *pluginsdk.PluginActionRequest) (*pluginsdk.PluginActionResponse, error) {
	switch request.ActionKey {
	case "connection.get":
		if connections, ok := w.resolver.(ConnectionSettingsStore); ok {
			settings, found, loadErr := connections.Load(ctx, request.Context.WorkspaceID)
			if loadErr != nil {
				return nil, unavailableActionError("load Bitbucket connection: %v", loadErr)
			}
			if !found {
				return actionResponse(map[string]any{"state": "unconfigured", "healthy": false})
			}
			oauthConfigured := settings.AuthMethod == "oauth" && settings.OAuthClientID != ""
			if inspector, ok := w.resolver.(interface {
				oauthRegistrationConfigured(context.Context, string) (bool, error)
			}); ok {
				oauthConfigured, _ = inspector.oauthRegistrationConfigured(ctx, request.Context.WorkspaceID)
			}
			provider, providerErr := w.provider(ctx, request.Context.WorkspaceID)
			if providerErr != nil {
				return actionResponse(connectionResponse(settings, false, providerErr, oauthConfigured))
			}
			healthErr := provider.Health(ctx)
			return actionResponse(connectionResponse(settings, healthErr == nil, healthErr, oauthConfigured))
		}
		provider, err := w.provider(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		err = provider.Health(ctx)
		return actionResponse(map[string]any{"healthy": err == nil, "capabilities": provider.Capabilities(), "error": safeError(err)})
	case "connection.disconnect":
		connections, ok := w.resolver.(interface {
			Disconnect(context.Context, string) error
		})
		if !ok {
			return nil, unavailableActionError("Bitbucket connection updates are unavailable")
		}
		if err := connections.Disconnect(ctx, request.Context.WorkspaceID); err != nil {
			return nil, unavailableActionError("disconnect Bitbucket connection: %v", err)
		}
		return actionResponse(map[string]any{"state": "unconfigured", "healthy": false})
	case "connection.save":
		var input connectionSaveInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		connections, ok := w.resolver.(ConnectionSettingsStore)
		if !ok {
			return nil, unavailableActionError("Bitbucket connection updates are unavailable")
		}
		settings, err := connections.Save(ctx, request.Context.WorkspaceID, input.ConnectionInput)
		if err != nil {
			if errors.Is(err, ErrInvalidConnectionInput) {
				return nil, pluginsdk.CategorizeActionError(pluginsdk.ActionErrorInvalidArgument, err)
			}
			return nil, unavailableActionError("save Bitbucket connection: %v", err)
		}
		if !input.Probe {
			return actionResponse(connectionResponse(settings, false, nil))
		}
		provider, err := w.provider(ctx, request.Context.WorkspaceID)
		if err != nil {
			return actionResponse(connectionResponse(settings, false, err))
		}
		err = provider.Health(ctx)
		return actionResponse(connectionResponse(settings, err == nil, err))
	case "oauth.start":
		coordinator, ok := w.resolver.(interface {
			StartOAuth(context.Context, string) (*url.URL, error)
		})
		if !ok {
			return nil, conflictActionError("Bitbucket OAuth is not configured")
		}
		authorizationURL, err := coordinator.StartOAuth(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		return actionResponse(map[string]any{"url": authorizationURL.String()})
	default:
		return nil, notFoundActionError("unsupported Bitbucket action %q", request.ActionKey)
	}
}
