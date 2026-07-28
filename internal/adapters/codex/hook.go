// Package codex translates supported Codex lifecycle hooks into the shared
// metadata-only event protocol.
package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

const (
	MaxHookInputBytes = 1 << 20
	MaxEventsPerHook  = 128
	DefaultDeadline   = 75 * time.Millisecond
)

type HookInput struct {
	SessionID      string          `json:"session_id"`
	CWD            string          `json:"cwd"`
	HookEventName  string          `json:"hook_event_name"`
	Source         string          `json:"source"`
	Reason         string          `json:"reason"`
	TurnID         string          `json:"turn_id"`
	AgentID        string          `json:"agent_id"`
	AgentType      string          `json:"agent_type"`
	ToolName       string          `json:"tool_name"`
	ToolUseID      string          `json:"tool_use_id"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response"`
	PermissionMode string          `json:"permission_mode"`
	Model          string          `json:"model"`
	TranscriptPath *string         `json:"transcript_path"`
}

type Sender interface {
	Send(context.Context, []protocol.Event) error
}

func Decode(reader io.Reader) (HookInput, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, MaxHookInputBytes+1))
	if err != nil {
		return HookInput{}, err
	}
	if len(payload) == 0 {
		return HookInput{}, errors.New("empty hook input")
	}
	if len(payload) > MaxHookInputBytes {
		return HookInput{}, errors.New("hook input exceeds limit")
	}
	var input HookInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return HookInput{}, err
	}
	if input.HookEventName == "" {
		return HookInput{}, errors.New("hook_event_name is required")
	}
	return input, nil
}

// Run is deliberately silent and failure-open. It never returns an error to a
// hook executable and never writes to an agent-owned output stream.
func Run(parent context.Context, reader io.Reader, sender Sender, now func() time.Time) {
	defer func() { _ = recover() }()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, DefaultDeadline)
	defer cancel()
	type decodeResult struct {
		input HookInput
		err   error
	}
	decoded := make(chan decodeResult, 1)
	go func() {
		defer func() { _ = recover() }()
		input, err := Decode(reader)
		decoded <- decodeResult{input: input, err: err}
	}()
	var result decodeResult
	select {
	case <-ctx.Done():
		return
	case result = <-decoded:
	}
	if result.err != nil {
		return
	}
	if now == nil {
		now = time.Now
	}
	events := Translate(result.input, now().UTC())
	if len(events) == 0 || sender == nil {
		return
	}
	sent := make(chan struct{}, 1)
	go func() {
		defer func() {
			_ = recover()
			sent <- struct{}{}
		}()
		_ = sender.Send(ctx, events)
	}()
	select {
	case <-ctx.Done():
	case <-sent:
	}
}
