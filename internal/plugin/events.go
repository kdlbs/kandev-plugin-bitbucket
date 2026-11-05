package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/kdlbs/kandev/pluginsdk"
)

// EventSink delegates plugin events to Kandev, which prefixes every name with
// plugin.kandev-plugin-bitbucket. Callers must pass only local event names.
type EventSink struct{ host pluginsdk.Host }

func NewEventSink(host pluginsdk.Host) (*EventSink, error) {
	if host == nil {
		return nil, fmt.Errorf("event host is required")
	}
	return &EventSink{host: host}, nil
}

func (s *EventSink) Emit(ctx context.Context, name string, payload map[string]any) error {
	if name == "" || strings.HasPrefix(name, "plugin.") {
		return fmt.Errorf("event name must be plugin-local")
	}
	return s.host.EmitEvent(ctx, name, payload)
}
