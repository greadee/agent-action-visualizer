package codex

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type captureSender struct {
	mu     sync.Mutex
	events []protocol.Event
	block  bool
	err    error
}

type noopSender struct{}

func (noopSender) Send(context.Context, []protocol.Event) error { return nil }

func (s *captureSender) Send(ctx context.Context, events []protocol.Event) error {
	if s.block {
		<-ctx.Done()
		return ctx.Err()
	}
	s.mu.Lock()
	s.events = append(s.events, events...)
	s.mu.Unlock()
	return s.err
}

func TestRunFailureOpenAndBounded(t *testing.T) {
	valid := `{"session_id":"session","hook_event_name":"SessionStart","source":"startup"}`
	for name, input := range map[string]string{
		"malformed":    `{`,
		"empty":        ``,
		"oversized":    strings.Repeat("x", MaxHookInputBytes+1),
		"disconnected": valid,
	} {
		t.Run(name, func(t *testing.T) {
			sender := &captureSender{err: errors.New("collector unavailable")}
			Run(context.Background(), strings.NewReader(input), sender, func() time.Time { return time.Unix(1, 0) })
		})
	}
	blocked := &captureSender{block: true}
	started := time.Now()
	Run(context.Background(), strings.NewReader(valid), blocked, func() time.Time { return time.Unix(1, 0) })
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("blocked collector delayed hook for %s", elapsed)
	}
	reader, writer := io.Pipe()
	started = time.Now()
	Run(context.Background(), reader, &captureSender{}, func() time.Time { return time.Unix(1, 0) })
	_ = reader.Close()
	_ = writer.Close()
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("blocked input delayed hook for %s", elapsed)
	}
}

func TestDecodeRejectsMissingEvent(t *testing.T) {
	if _, err := Decode(strings.NewReader(`{"session_id":"session"}`)); err == nil {
		t.Fatal("expected missing hook_event_name to fail")
	}
}

func BenchmarkRunHook(b *testing.B) {
	payload := `{"session_id":"session","cwd":"/workspace","hook_event_name":"PostToolUse","turn_id":"turn","tool_name":"apply_patch","tool_use_id":"tool","tool_input":{"command":"*** Begin Patch\n*** Update File: src/a.go\n@@\n-old\n+new\n*** End Patch"},"tool_response":"<redacted>"}`
	now := func() time.Time { return time.Unix(1, 0) }
	b.ReportAllocs()
	for range b.N {
		Run(context.Background(), strings.NewReader(payload), noopSender{}, now)
	}
}
