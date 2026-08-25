// Package example contains a small metadata-only adapter implementation.
package example

import (
	"context"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

// Static emits a fixed event batch. It is intended as a contract example and
// test fixture, not as a production observation source.
type Static struct {
	Events []protocol.Event
}

func (Static) Descriptor() adapter.Descriptor {
	return adapter.Descriptor{
		ID:              "example.static",
		Version:         adapter.SDKVersion,
		ProtocolVersion: protocol.SchemaVersion,
		Capabilities: []adapter.Capability{
			adapter.CapabilitySessionLifecycle,
			adapter.CapabilityFileRead,
		},
	}
}

func (s Static) Observe(ctx context.Context, emitter adapter.Emitter) {
	if emitter == nil {
		return
	}
	emitter.Emit(ctx, append([]protocol.Event(nil), s.Events...))
}
