package plugin

import (
	"context"

	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

func (w *Workflows) handleWatchAction(ctx context.Context, request *pluginsdk.PluginActionRequest) (*pluginsdk.PluginActionResponse, error) {
	switch request.ActionKey {
	case "watches.get":
		result, err := w.watches.List(ctx, request.Context.WorkspaceID)
		if err != nil {
			return nil, err
		}
		return actionResponse(map[string]any{"watches": result})
	case "watches.create":
		var watch watches.Watch
		if err := decodeAction(request.Body, &watch); err != nil {
			return nil, err
		}
		watch.WorkspaceID = request.Context.WorkspaceID
		result, err := w.watches.Create(ctx, watch)
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	case "watches.update":
		var input watchUpdateInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		watchID := input.WatchID
		if watchID == "" {
			watchID = "default"
		}
		watch, err := w.watches.Get(ctx, request.Context.WorkspaceID, watchID)
		if err == watches.ErrWatchNotFound {
			if !input.Enabled {
				return actionResponse(map[string]any{"watches": []watches.Watch{}})
			}
			watch, err = w.watches.Create(ctx, watches.Watch{ID: watchID, WorkspaceID: request.Context.WorkspaceID, Filter: input.Filter, Launch: input.Launch})
		}
		if err != nil {
			return nil, err
		}
		if hasFilter(input.Filter) {
			watch, err = w.watches.SetFilter(ctx, request.Context.WorkspaceID, watchID, input.Filter)
			if err != nil {
				return nil, err
			}
		}
		if input.Enabled {
			watch, err = w.watches.Resume(ctx, request.Context.WorkspaceID, watchID)
		} else {
			watch, err = w.watches.Pause(ctx, request.Context.WorkspaceID, watchID)
		}
		if err != nil {
			return nil, err
		}
		return actionResponse(map[string]any{"watch": watch})
	case "watches.filter":
		var input watchFilterInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		result, err := w.watches.SetFilter(ctx, request.Context.WorkspaceID, input.WatchID, input.Filter)
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	case "watches.preset":
		var input watchPresetInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		result, err := w.watches.SavePreset(ctx, request.Context.WorkspaceID, input.WatchID, input.Preset)
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	case "watches.run":
		var input watchIDInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		result, err := w.watches.Run(ctx, request.Context.WorkspaceID, input.WatchID)
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	case "watches.pause", "watches.resume":
		var input watchIDInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		var result watches.Watch
		var err error
		if request.ActionKey == "watches.pause" {
			result, err = w.watches.Pause(ctx, request.Context.WorkspaceID, input.WatchID)
		} else {
			result, err = w.watches.Resume(ctx, request.Context.WorkspaceID, input.WatchID)
		}
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	case "watches.preview_reset", "watches.preview_delete":
		var input watchIDInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		if request.ActionKey == "watches.preview_reset" {
			result, err := w.watches.PreviewReset(ctx, request.Context.WorkspaceID, input.WatchID)
			if err != nil {
				return nil, err
			}
			return actionResponse(result)
		}
		result, err := w.watches.PreviewDelete(ctx, request.Context.WorkspaceID, input.WatchID)
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	case "watches.reset", "watches.delete":
		var input watchIDInput
		if err := decodeAction(request.Body, &input); err != nil {
			return nil, err
		}
		if request.ActionKey == "watches.reset" {
			result, err := w.watches.Reset(ctx, request.Context.WorkspaceID, input.WatchID)
			if err != nil {
				return nil, err
			}
			return actionResponse(result)
		}
		result, err := w.watches.Delete(ctx, request.Context.WorkspaceID, input.WatchID)
		if err != nil {
			return nil, err
		}
		return actionResponse(result)
	default:
		return nil, notFoundActionError("unsupported Bitbucket action %q", request.ActionKey)
	}
}
