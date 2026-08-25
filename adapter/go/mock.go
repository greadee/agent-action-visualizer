package adapter

import (
	"context"
	"sync"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

// MockCollector is an in-memory Collector for adapter contract and integration
// tests. Set Err or Block before use to exercise failure-open behavior.
type MockCollector struct {
	mu     sync.Mutex
	events []protocol.Event
	calls  int

	Err   error
	Block <-chan struct{}
}

func (m *MockCollector) Send(ctx context.Context, events []protocol.Event) error {
	if m.Block != nil {
		select {
		case <-m.Block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.events = append(m.events, events...)
	return m.Err
}

func (m *MockCollector) Events() []protocol.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]protocol.Event(nil), m.events...)
}

func (m *MockCollector) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}
