// Package adapter provides the public contract for local AAV adapters.
package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

// SDKVersion changes only when this public adapter contract changes.
const SDKVersion = "1.0"

type Capability string

const (
	CapabilitySessionLifecycle Capability = "session_lifecycle"
	CapabilityAgentLifecycle   Capability = "agent_lifecycle"
	CapabilityToolLifecycle    Capability = "tool_lifecycle"
	CapabilityFileRead         Capability = "file_read"
	CapabilityFileWrite        Capability = "file_write"
	CapabilityFileMove         Capability = "file_move"
	CapabilityFileDelete       Capability = "file_delete"
	CapabilityCommandLifecycle Capability = "command_lifecycle"
	CapabilityStructuredPatch  Capability = "structured_patch"
	CapabilityWorkDelta        Capability = "work_delta"
)

var validCapabilities = map[Capability]bool{
	CapabilitySessionLifecycle: true,
	CapabilityAgentLifecycle:   true,
	CapabilityToolLifecycle:    true,
	CapabilityFileRead:         true,
	CapabilityFileWrite:        true,
	CapabilityFileMove:         true,
	CapabilityFileDelete:       true,
	CapabilityCommandLifecycle: true,
	CapabilityStructuredPatch:  true,
	CapabilityWorkDelta:        true,
}

// Descriptor declares an adapter's stable identity and evidence it can emit.
type Descriptor struct {
	ID              string       `json:"id"`
	Version         string       `json:"version"`
	ProtocolVersion string       `json:"protocol_version"`
	Capabilities    []Capability `json:"capabilities"`
}

func (d Descriptor) Validate() error {
	if d.ID == "" {
		return errors.New("adapter id is required")
	}
	if d.Version == "" {
		return errors.New("adapter version is required")
	}
	if d.ProtocolVersion != protocol.SchemaVersion {
		return fmt.Errorf("unsupported protocol_version %q", d.ProtocolVersion)
	}
	seen := make(map[Capability]bool, len(d.Capabilities))
	for _, capability := range d.Capabilities {
		if !validCapabilities[capability] {
			return fmt.Errorf("unsupported capability %q", capability)
		}
		if seen[capability] {
			return fmt.Errorf("duplicate capability %q", capability)
		}
		seen[capability] = true
	}
	return nil
}

// Emitter accepts normalized event envelopes. Emit is deliberately errorless:
// adapters must not allow collector availability to affect an observed agent.
type Emitter interface {
	Emit(context.Context, []protocol.Event)
}

// Adapter is the public implementation contract. Observe must be silent and
// failure-open: it may lose visualization data, but must not change agent
// output, working files, exit status, or control flow.
type Adapter interface {
	Descriptor() Descriptor
	Observe(context.Context, Emitter)
}

// ValidateAdapter verifies the static portion of the public adapter contract.
func ValidateAdapter(candidate Adapter) error {
	if candidate == nil {
		return errors.New("adapter is required")
	}
	return candidate.Descriptor().Validate()
}

// Collector is the local transport boundary used by BoundedEmitter.
// Collectors must treat supplied events as metadata-only protocol envelopes.
type Collector interface {
	Send(context.Context, []protocol.Event) error
}

type EmitterConfig struct {
	// MaxInFlight bounds send goroutines. Full capacity drops the new batch.
	MaxInFlight int
	// MaxEvents bounds events accepted from one observation. Excess events drop.
	MaxEvents int
	// Deadline bounds a collector call. Zero selects DefaultEmitterDeadline.
	Deadline time.Duration
}

const (
	DefaultMaxInFlight     = 4
	DefaultMaxEvents       = 128
	DefaultEmitterDeadline = 75 * time.Millisecond
)

// BoundedEmitter is a non-blocking, failure-open adapter-to-collector bridge.
// Invalid batches and unavailable collectors are silently dropped.
type BoundedEmitter struct {
	collector Collector
	deadline  time.Duration
	maxEvents int
	slots     chan struct{}
}

func NewBoundedEmitter(collector Collector, config EmitterConfig) *BoundedEmitter {
	if config.MaxInFlight <= 0 {
		config.MaxInFlight = DefaultMaxInFlight
	}
	if config.MaxEvents <= 0 {
		config.MaxEvents = DefaultMaxEvents
	}
	if config.Deadline <= 0 {
		config.Deadline = DefaultEmitterDeadline
	}
	return &BoundedEmitter{
		collector: collector,
		deadline:  config.Deadline,
		maxEvents: config.MaxEvents,
		slots:     make(chan struct{}, config.MaxInFlight),
	}
}

func (e *BoundedEmitter) Emit(parent context.Context, events []protocol.Event) {
	if e == nil || e.collector == nil {
		return
	}
	batch := validBatch(events, e.maxEvents)
	if len(batch) == 0 {
		return
	}
	select {
	case e.slots <- struct{}{}:
	default:
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, e.deadline)
	go func() {
		defer func() {
			_ = recover()
			cancel()
			<-e.slots
		}()
		_ = e.collector.Send(ctx, batch)
	}()
}

func validBatch(events []protocol.Event, limit int) []protocol.Event {
	if len(events) > limit {
		events = events[:limit]
	}
	batch := make([]protocol.Event, 0, len(events))
	for _, event := range events {
		if event.Validate() == nil {
			batch = append(batch, event)
		}
	}
	return batch
}
