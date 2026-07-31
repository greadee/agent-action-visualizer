package wrapper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	"github.com/greadee/agent-action-visualizer/adapter/go/wrapper"
	filesystemadapter "github.com/greadee/agent-action-visualizer/internal/adapters/filesystem"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestDescriptorSatisfiesAdapterContract(t *testing.T) {
	descriptor := wrapper.Descriptor()
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	if descriptor.ID != wrapper.AdapterID || len(descriptor.Capabilities) != 5 {
		t.Fatalf("descriptor = %#v", descriptor)
	}
}

func TestRunPreservesArgumentsEnvironmentAndIO(t *testing.T) {
	collector := &adapter.MockCollector{}
	var stdout, stderr bytes.Buffer
	input := "stdin with spaces and \"quotes\"\n"
	arguments := []string{"plain", "space value", `quote"value`, "--leading"}
	secret := `environment with spaces and "quotes"`
	config := helperConfig(t, "inspect", arguments...)
	config.Env = setEnvironment(config.Env, "AAV_WRAPPER_TEST_VALUE", secret)
	config.Stdin = strings.NewReader(input)
	config.Stdout = &stdout
	config.Stderr = &stderr
	config.Collector = collector
	config.SessionID = "session-inspect"

	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 0 || result.Signal != "" || result.StartError != nil {
		t.Fatalf("result = %#v", result)
	}
	var observed helperObservation
	if err := json.Unmarshal(stdout.Bytes(), &observed); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if strings.Join(observed.Args, "\x00") != strings.Join(arguments, "\x00") ||
		observed.Environment != secret || observed.Stdin != input ||
		filepath.Clean(observed.WorkingDirectory) != filepath.Clean(config.Dir) {
		t.Fatalf("observation = %#v", observed)
	}
	if stderr.String() != "helper stderr\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}

	events := collector.Events()
	if len(events) != 4 {
		t.Fatalf("lifecycle events = %#v", events)
	}
	payload, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte(secret)) || bytes.Contains(payload, []byte("space value")) {
		t.Fatalf("lifecycle persisted arguments or environment: %s", payload)
	}
	if events[0].EventType != protocol.EventSessionStarted ||
		events[1].EventType != protocol.EventCommandStarted ||
		events[2].EventType != protocol.EventCommandCompleted ||
		events[3].EventType != protocol.EventSessionStopped {
		t.Fatalf("lifecycle order = %#v", events)
	}
}

func TestRunPreservesFailureExitAndOutput(t *testing.T) {
	collector := &adapter.MockCollector{}
	var stdout, stderr bytes.Buffer
	config := helperConfig(t, "exit", "23")
	config.Stdout = &stdout
	config.Stderr = &stderr
	config.Collector = collector
	config.SessionID = "session-failure"

	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 23 || result.StartError != nil {
		t.Fatalf("result = %#v", result)
	}
	if stdout.String() != "failure stdout\n" || stderr.String() != "failure stderr\n" {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	events := collector.Events()
	if len(events) != 4 || events[2].Status != "failed" || events[3].Status != "failed" {
		t.Fatalf("events = %#v", events)
	}
	if exitCode, ok := events[3].Metadata["exit_code"].(int); !ok || exitCode != 23 {
		t.Fatalf("exit metadata = %#v", events[3].Metadata)
	}
}

func TestRunPreservesLargeOutput(t *testing.T) {
	const size = 2 << 20
	var stdout, stderr bytes.Buffer
	config := helperConfig(t, "large", strconv.Itoa(size))
	config.Stdout = &stdout
	config.Stderr = &stderr

	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 0 {
		t.Fatalf("result = %#v", result)
	}
	if stdout.Len() != size || stderr.Len() != size {
		t.Fatalf("large output stdout=%d stderr=%d", stdout.Len(), stderr.Len())
	}
	if !bytes.Equal(stdout.Bytes(), bytes.Repeat([]byte("o"), size)) ||
		!bytes.Equal(stderr.Bytes(), bytes.Repeat([]byte("e"), size)) {
		t.Fatal("large output bytes changed")
	}
}

func TestDisconnectedCollectorDoesNotChangeCommand(t *testing.T) {
	collector := blockingCollector{}
	var stdout bytes.Buffer
	config := helperConfig(t, "success")
	config.Stdout = &stdout
	config.Collector = collector
	config.SendDeadline = 20 * time.Millisecond

	started := time.Now()
	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 0 || stdout.String() != "success\n" {
		t.Fatalf("result=%#v stdout=%q", result, stdout.String())
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("collector delayed command for %s", elapsed)
	}
}

func TestStructuredParserAndFallbackAreAsynchronous(t *testing.T) {
	collector := &adapter.MockCollector{}
	parser := &recordingParser{}
	fallback := &recordingFallback{called: make(chan wrapper.Observation, 1)}
	var stdout bytes.Buffer
	config := helperConfig(t, "stream")
	config.Stdout = &stdout
	config.Collector = collector
	config.Parser = parser
	config.Fallback = fallback
	config.SessionID = "session-stream"

	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 0 || stdout.String() != "structured record\n" {
		t.Fatalf("result=%#v stdout=%q", result, stdout.String())
	}
	select {
	case observation := <-fallback.called:
		if observation.SessionID != "session-stream" || observation.ProcessID == 0 {
			t.Fatalf("fallback observation = %#v", observation)
		}
	default:
		t.Fatal("fallback observer was not started")
	}
	events := collector.Events()
	found := false
	for _, event := range events {
		if event.EventID == "stream-event" {
			found = true
		}
	}
	if !found {
		t.Fatalf("parser event missing from %#v", events)
	}
}

func TestBlockedParserAndFallbackDoNotDelayCommand(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	var stdout bytes.Buffer
	config := helperConfig(t, "stream")
	config.Stdout = &stdout
	config.Parser = blockingParser{release: release}
	config.Fallback = blockingFallback{release: release}
	config.ParserDrain = 15 * time.Millisecond

	started := time.Now()
	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 0 || stdout.String() != "structured record\n" {
		t.Fatalf("result=%#v stdout=%q", result, stdout.String())
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("observer delayed command for %s", elapsed)
	}
}

func TestRunObservesChildFileWriteWithoutChangingFile(t *testing.T) {
	collector := &adapter.MockCollector{}
	config := helperConfig(t, "write")
	path := filepath.Join(config.Dir, "child.txt")
	config.Env = setEnvironment(config.Env, "AAV_WRAPPER_TEST_PATH", path)
	config.Collector = collector
	config.SessionID = "session-write"
	config.Fallback = filesystemadapter.NewObserver(filesystemadapter.Config{
		Debounce:     10 * time.Millisecond,
		RenameWindow: 20 * time.Millisecond,
	})

	result := wrapper.Run(context.Background(), config)
	if result.ExitCode != 0 || result.StartError != nil {
		t.Fatalf("result = %#v", result)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "child-owned content\n" {
		t.Fatalf("content=%q err=%v", content, err)
	}
	events := collector.Events()
	found := false
	for _, event := range events {
		if event.Path == path &&
			(event.EventType == protocol.EventFileCreated || event.EventType == protocol.EventFileModified) {
			found = true
		}
	}
	if !found {
		t.Fatalf("filesystem event missing from %#v", events)
	}
}

func TestContextCancellationStopsChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	config := helperConfig(t, "sleep")
	started := time.Now()
	result := wrapper.Run(ctx, config)
	if result.ExitCode == 0 {
		t.Fatalf("canceled result = %#v", result)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("canceled child took %s", elapsed)
	}
}

func TestStartFailureIsLocal(t *testing.T) {
	collector := &adapter.MockCollector{}
	result := wrapper.Run(context.Background(), wrapper.Config{
		Command:   "aav-command-that-does-not-exist",
		Collector: collector,
	})
	if result.ExitCode != 127 || result.StartError == nil || len(collector.Events()) != 0 {
		t.Fatalf("result=%#v events=%#v", result, collector.Events())
	}
}

type helperObservation struct {
	Args             []string `json:"args"`
	Environment      string   `json:"environment"`
	Stdin            string   `json:"stdin"`
	WorkingDirectory string   `json:"working_directory"`
}

func helperConfig(t *testing.T, mode string, arguments ...string) wrapper.Config {
	t.Helper()
	args := []string{"-test.run=TestWrapperChildHelper", "--"}
	args = append(args, arguments...)
	return wrapper.Config{
		Command: os.Args[0],
		Args:    args,
		Env:     setEnvironment(os.Environ(), "AAV_WRAPPER_CHILD_HELPER", mode),
		Dir:     t.TempDir(),
	}
}

func setEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func TestWrapperChildHelper(t *testing.T) {
	mode := os.Getenv("AAV_WRAPPER_CHILD_HELPER")
	if mode == "" {
		return
	}
	arguments := argumentsAfterSeparator(os.Args)
	switch mode {
	case "inspect":
		input, _ := io.ReadAll(os.Stdin)
		workingDirectory, _ := os.Getwd()
		_ = json.NewEncoder(os.Stdout).Encode(helperObservation{
			Args:             arguments,
			Environment:      os.Getenv("AAV_WRAPPER_TEST_VALUE"),
			Stdin:            string(input),
			WorkingDirectory: workingDirectory,
		})
		_, _ = fmt.Fprintln(os.Stderr, "helper stderr")
	case "exit":
		_, _ = fmt.Fprintln(os.Stdout, "failure stdout")
		_, _ = fmt.Fprintln(os.Stderr, "failure stderr")
		code, _ := strconv.Atoi(arguments[0])
		os.Exit(code)
	case "large":
		size, _ := strconv.Atoi(arguments[0])
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("o"), size))
		_, _ = os.Stderr.Write(bytes.Repeat([]byte("e"), size))
	case "stream":
		_, _ = fmt.Fprintln(os.Stdout, "structured record")
		time.Sleep(25 * time.Millisecond)
	case "sleep":
		time.Sleep(10 * time.Second)
	case "success":
		_, _ = fmt.Fprintln(os.Stdout, "success")
	case "write":
		_ = os.WriteFile(os.Getenv("AAV_WRAPPER_TEST_PATH"), []byte("child-owned content\n"), 0o600)
		time.Sleep(200 * time.Millisecond)
	default:
		os.Exit(64)
	}
	os.Exit(0)
}

func argumentsAfterSeparator(arguments []string) []string {
	for index, value := range arguments {
		if value == "--" {
			return append([]string(nil), arguments[index+1:]...)
		}
	}
	return nil
}

type blockingCollector struct{}

func (blockingCollector) Send(ctx context.Context, _ []protocol.Event) error {
	<-ctx.Done()
	return errors.New("collector unavailable")
}

type recordingParser struct {
	mu      sync.Mutex
	emitted bool
}

func (p *recordingParser) Parse(_ context.Context, chunk wrapper.StreamChunk) []protocol.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.emitted || chunk.EOF || !bytes.Contains(chunk.Data, []byte("structured record")) {
		return nil
	}
	p.emitted = true
	return []protocol.Event{{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          "stream-event",
		SessionID:        "session-stream",
		AdapterID:        "test-parser",
		AdapterVersion:   "1",
		SourceType:       protocol.SourceStructuredStream,
		SourceConfidence: protocol.ConfidenceExact,
		EventType:        protocol.EventToolCompleted,
		Timestamp:        time.Unix(1, 0).UTC(),
	}}
}

type blockingParser struct{ release <-chan struct{} }

func (p blockingParser) Parse(context.Context, wrapper.StreamChunk) []protocol.Event {
	<-p.release
	return nil
}

type recordingFallback struct {
	called chan wrapper.Observation
}

func (f *recordingFallback) Observe(ctx context.Context, observation wrapper.Observation, _ adapter.Emitter) {
	f.called <- observation
	<-ctx.Done()
}

type blockingFallback struct{ release <-chan struct{} }

func (f blockingFallback) Observe(context.Context, wrapper.Observation, adapter.Emitter) {
	<-f.release
}
