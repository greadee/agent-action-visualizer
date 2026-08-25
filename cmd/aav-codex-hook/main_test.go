package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/greadee/agent-action-visualizer/internal/ipc"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestHookProcessPreservesExitAndOutput(t *testing.T) {
	endpoint := filepath.Join(t.TempDir(), "missing.sock")
	if runtime.GOOS == "windows" {
		endpoint = hookTestEndpoint(t.Name() + "-missing")
	}
	command := exec.Command(os.Args[0], "-test.run=TestHookHelperProcess")
	command.Env = append(os.Environ(), "AAV_HOOK_HELPER=1", "AAV_COLLECTOR_ENDPOINT="+endpoint)
	command.Stdin = strings.NewReader(`{"session_id":"session","hook_event_name":"SessionStart","source":"startup"}`)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("hook changed exit behavior: %v output=%q", err, output)
	}
	if len(output) != 0 {
		t.Fatalf("hook wrote agent-visible output: %q", output)
	}
}

func TestHookProcessSubmitsToLocalCollector(t *testing.T) {
	endpoint := hookTestEndpoint(t.Name())
	received := make(chan protocol.Event, 4)
	server := ipc.NewServer(endpoint, func(event protocol.Event) bool {
		received <- event
		return true
	})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	command := exec.Command(os.Args[0], "-test.run=TestHookHelperProcess")
	command.Env = append(os.Environ(), "AAV_HOOK_HELPER=1", "AAV_COLLECTOR_ENDPOINT="+endpoint)
	command.Stdin = strings.NewReader(`{"session_id":"session","cwd":"C:\\workspace","hook_event_name":"PostToolUse","turn_id":"turn","tool_name":"apply_patch","tool_use_id":"tool","tool_input":{"command":"*** Begin Patch\n*** Update File: main.go\n@@\n-old\n+new\n*** End Patch"},"tool_response":"<redacted>"}`)
	output, err := command.CombinedOutput()
	if err != nil || len(output) != 0 {
		t.Fatalf("hook process err=%v output=%q", err, output)
	}
	var events []protocol.Event
	deadline := time.After(time.Second)
	for len(events) < 2 {
		select {
		case event := <-received:
			events = append(events, event)
		case <-deadline:
			t.Fatalf("received events = %#v", events)
		}
	}
	if events[0].EventType != protocol.EventToolCompleted || events[1].EventType != protocol.EventFilePatched || events[1].LinesAdded == nil || *events[1].LinesAdded != 1 {
		t.Fatalf("normalized process events = %#v", events)
	}
}

func TestHookHelperProcess(t *testing.T) {
	if os.Getenv("AAV_HOOK_HELPER") != "1" {
		return
	}
	main()
	os.Exit(0)
}

func BenchmarkHookProcessRoundTrip(b *testing.B) {
	endpoint := hookTestEndpoint(b.Name())
	server := ipc.NewServer(endpoint, func(protocol.Event) bool { return true })
	if err := server.Start(); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(server.Close)
	payload := `{"session_id":"session","hook_event_name":"SessionStart","source":"startup"}`
	environment := append(os.Environ(), "AAV_HOOK_HELPER=1", "AAV_COLLECTOR_ENDPOINT="+endpoint)
	b.ResetTimer()
	for range b.N {
		command := exec.Command(os.Args[0], "-test.run=TestHookHelperProcess")
		command.Env = environment
		command.Stdin = strings.NewReader(payload)
		output, err := command.CombinedOutput()
		if err != nil || len(output) != 0 {
			b.Fatalf("hook process err=%v output=%q", err, output)
		}
	}
}

func hookTestEndpoint(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	if runtime.GOOS == "windows" {
		return `\\.\pipe\aav-hook-test-` + hex.EncodeToString(sum[:8])
	}
	return filepath.Join(os.TempDir(), "aav-hook-test-"+hex.EncodeToString(sum[:8])+".sock")
}
