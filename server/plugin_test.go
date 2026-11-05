package main

import (
	"context"
	"testing"

	"github.com/kdlbs/kandev/pluginsdk"
	"github.com/stretchr/testify/require"
)

func TestBitbucketPluginDoesNotUseStateBeforeBehaviorExists(t *testing.T) {
	p := &bitbucketPlugin{}
	host := newStateRecordingHost()
	p.SetHost(host)

	require.NoError(t, p.OnEvent(context.Background(), &pluginsdk.Event{
		EventID:   "event-1",
		EventType: "task.created",
	}))
	require.Empty(t, host.state)
}

type stateRecordingHost struct {
	pluginsdk.UnimplementedHostData
	state map[string]map[string]any
}

func newStateRecordingHost() *stateRecordingHost {
	return &stateRecordingHost{state: make(map[string]map[string]any)}
}

func (h *stateRecordingHost) GetState(context.Context, string, string, string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (h *stateRecordingHost) SetState(_ context.Context, scope, scopeID, key string, value map[string]any) error {
	h.state[scope+"/"+scopeID+"/"+key] = value
	return nil
}

func (h *stateRecordingHost) DeleteState(context.Context, string, string, string) error { return nil }

func (h *stateRecordingHost) ListState(context.Context, string, string) ([]pluginsdk.StateEntry, error) {
	return nil, nil
}

func (h *stateRecordingHost) GetConfig(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}

func (h *stateRecordingHost) RevealSecret(context.Context, string) (string, error) { return "", nil }

func (h *stateRecordingHost) GetSecret(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (h *stateRecordingHost) SetSecret(context.Context, string, string) error { return nil }

func (h *stateRecordingHost) DeleteSecret(context.Context, string) error { return nil }

func (h *stateRecordingHost) EmitEvent(context.Context, string, map[string]any) error { return nil }

var _ pluginsdk.Host = (*stateRecordingHost)(nil)
