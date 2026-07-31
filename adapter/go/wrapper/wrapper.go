// Package wrapper runs a command while emitting local, metadata-only lifecycle
// events without changing the command's arguments, environment, or I/O.
package wrapper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

const (
	AdapterID      = "generic-wrapper"
	AdapterVersion = adapter.SDKVersion

	DefaultDeliveryQueue = 64
	DefaultSendDeadline  = 75 * time.Millisecond
	DefaultParserQueue   = 64
	DefaultParserDrain   = 25 * time.Millisecond
	DefaultFallbackDrain = 25 * time.Millisecond
)

type Stream string

const (
	StreamStdout Stream = "stdout"
	StreamStderr Stream = "stderr"
)

// StreamChunk is a bounded copy of child output. Data is observational only;
// the original bytes are written to the child-owned destination first.
type StreamChunk struct {
	Stream        Stream
	Data          []byte
	Sequence      uint64
	DroppedBefore bool
	EOF           bool
}

// StructuredStreamParser incrementally translates an explicitly supported
// local stream. Implementations must tolerate arbitrary chunk boundaries and
// reset partial framing when DroppedBefore is true.
type StructuredStreamParser interface {
	Parse(context.Context, StreamChunk) []protocol.Event
}

type Observation struct {
	SessionID   string
	ProjectRoot string
	AgentType   string
	ProcessID   int
}

// FallbackObserver is the integration point implemented by P7-S3. Observe is
// always called asynchronously and is canceled when the child exits.
type FallbackObserver interface {
	Observe(context.Context, Observation, adapter.Emitter)
}

type Config struct {
	Command string
	Args    []string
	Env     []string
	Dir     string

	ProjectRoot string
	SessionID   string
	AgentType   string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Collector adapter.Collector
	Parser    StructuredStreamParser
	Fallback  FallbackObserver

	DeliveryQueue  int
	SendDeadline   time.Duration
	ParserQueue    int
	ParserDrain    time.Duration
	FallbackDrain  time.Duration
	EvidenceQueue  int
	EvidenceWindow time.Duration
	Now            func() time.Time
}

type Result struct {
	ExitCode   int
	Signal     string
	StartError error

	processSignal os.Signal
}

func Descriptor() adapter.Descriptor {
	return adapter.Descriptor{
		ID:              AdapterID,
		Version:         AdapterVersion,
		ProtocolVersion: protocol.SchemaVersion,
		Capabilities: []adapter.Capability{
			adapter.CapabilitySessionLifecycle,
			adapter.CapabilityCommandLifecycle,
			adapter.CapabilityFileWrite,
			adapter.CapabilityFileMove,
			adapter.CapabilityFileDelete,
		},
	}
}

func Run(parent context.Context, config Config) Result {
	if parent == nil {
		parent = context.Background()
	}
	prepared, err := prepareConfig(config)
	if err != nil {
		return Result{ExitCode: 127, StartError: err}
	}

	dispatch := newDispatcher(prepared.Collector, prepared.DeliveryQueue, prepared.SendDeadline)
	defer dispatch.stop()
	evidence := newEvidenceGate(dispatch, prepared.EvidenceQueue, prepared.EvidenceWindow)
	defer evidence.stop(prepared.ParserDrain)
	parserTap := newStreamTap(parent, prepared.Parser, evidence, prepared.ParserQueue)

	command := exec.Command(prepared.Command, prepared.Args...)
	command.Env = prepared.Env
	command.Dir = prepared.Dir
	command.Stdin = prepared.Stdin
	command.Stdout = observedWriter(prepared.Stdout, StreamStdout, parserTap)
	command.Stderr = observedWriter(prepared.Stderr, StreamStderr, parserTap)
	configureProcess(command)

	if err := command.Start(); err != nil {
		parserTap.finish(prepared.ParserDrain)
		return Result{ExitCode: 127, StartError: err}
	}

	startedAt := prepared.Now().UTC()
	observation := Observation{
		SessionID:   prepared.SessionID,
		ProjectRoot: prepared.ProjectRoot,
		AgentType:   prepared.AgentType,
		ProcessID:   command.Process.Pid,
	}
	dispatch.emit(lifecycleStarted(observation, prepared.Command, startedAt))
	parserTap.start()

	observerCtx, cancelObservers := context.WithCancel(parent)
	observerDone := make(chan struct{})
	if prepared.Fallback != nil {
		go func() {
			defer close(observerDone)
			observeFallback(observerCtx, prepared.Fallback, observation, evidence)
		}()
	} else {
		close(observerDone)
	}

	processDone := make(chan struct{})
	stopSignals := relaySignals(command.Process)
	go cancelProcess(parent, command.Process, processDone)
	waitErr := command.Wait()
	close(processDone)
	stopSignals()
	cancelObservers()
	waitForObserver(observerDone, prepared.FallbackDrain)
	parserTap.finish(prepared.ParserDrain)
	evidence.stop(prepared.ParserDrain)

	result := processResult(waitErr, command.ProcessState)
	stoppedAt := prepared.Now().UTC()
	finalAck := dispatch.emitFinal(lifecycleStopped(observation, prepared.Command, startedAt, stoppedAt, result))
	dispatch.flush(finalAck, 2*prepared.SendDeadline)
	return result
}

func prepareConfig(config Config) (Config, error) {
	if strings.TrimSpace(config.Command) == "" {
		return Config{}, errors.New("command is required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.SessionID == "" {
		config.SessionID = newSessionID(config.Now())
	}
	if config.AgentType == "" {
		config.AgentType = "generic"
	}
	if config.Dir == "" {
		current, err := os.Getwd()
		if err != nil {
			return Config{}, fmt.Errorf("resolve working directory: %w", err)
		}
		config.Dir = current
	}
	dir, err := filepath.Abs(config.Dir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve working directory: %w", err)
	}
	config.Dir = dir
	if config.ProjectRoot == "" {
		config.ProjectRoot = config.Dir
	}
	root, err := filepath.Abs(config.ProjectRoot)
	if err != nil {
		return Config{}, fmt.Errorf("resolve project root: %w", err)
	}
	config.ProjectRoot = root
	if config.Stdout == nil {
		config.Stdout = io.Discard
	}
	if config.Stderr == nil {
		config.Stderr = io.Discard
	}
	if config.DeliveryQueue <= 0 {
		config.DeliveryQueue = DefaultDeliveryQueue
	}
	if config.SendDeadline <= 0 {
		config.SendDeadline = DefaultSendDeadline
	}
	if config.ParserQueue <= 0 {
		config.ParserQueue = DefaultParserQueue
	}
	if config.ParserDrain <= 0 {
		config.ParserDrain = DefaultParserDrain
	}
	if config.FallbackDrain <= 0 {
		config.FallbackDrain = DefaultFallbackDrain
	}
	if config.EvidenceQueue <= 0 {
		config.EvidenceQueue = DefaultEvidenceQueue
	}
	if config.EvidenceWindow <= 0 {
		config.EvidenceWindow = DefaultEvidenceWindow
	}
	config.Args = append([]string(nil), config.Args...)
	if config.Env != nil {
		config.Env = append([]string(nil), config.Env...)
	}
	return config, nil
}

func newSessionID(now time.Time) string {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err == nil {
		return "wrapper-" + hex.EncodeToString(random)
	}
	return fmt.Sprintf("wrapper-%d-%d", now.UTC().UnixNano(), os.Getpid())
}

func observeFallback(ctx context.Context, observer FallbackObserver, observation Observation, emitter adapter.Emitter) {
	defer func() { _ = recover() }()
	observer.Observe(ctx, observation, emitter)
}

func waitForObserver(done <-chan struct{}, deadline time.Duration) {
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
	}
}

func cancelProcess(ctx context.Context, process *os.Process, done <-chan struct{}) {
	select {
	case <-ctx.Done():
		interruptProcess(process)
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			killProcess(process)
		}
	case <-done:
	}
}
