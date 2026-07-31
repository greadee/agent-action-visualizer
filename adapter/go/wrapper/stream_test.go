package wrapper

import (
	"context"
	"sync"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestStreamTapReportsDroppedChunksAtEOF(t *testing.T) {
	parser := &blockingRecordingParser{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	dispatch := newDispatcher(nil, 4, time.Millisecond)
	defer dispatch.stop()
	tap := newStreamTap(context.Background(), parser, dispatch, 1)
	tap.start()
	tap.offer(StreamStdout, []byte("first"))
	select {
	case <-parser.entered:
	case <-time.After(time.Second):
		t.Fatal("parser did not receive first chunk")
	}
	tap.offer(StreamStdout, []byte("second"))
	tap.offer(StreamStdout, []byte("dropped"))
	close(parser.release)
	tap.finish(time.Second)

	chunks := parser.snapshot()
	if len(chunks) < 3 {
		t.Fatalf("chunks = %#v", chunks)
	}
	stdoutEOF := chunks[len(chunks)-2]
	if !stdoutEOF.EOF || stdoutEOF.Stream != StreamStdout || !stdoutEOF.DroppedBefore {
		t.Fatalf("stdout EOF = %#v", stdoutEOF)
	}
}

type blockingRecordingParser struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	chunks  []StreamChunk
}

func (p *blockingRecordingParser) Parse(_ context.Context, chunk StreamChunk) []protocol.Event {
	p.once.Do(func() {
		close(p.entered)
		<-p.release
	})
	p.mu.Lock()
	p.chunks = append(p.chunks, chunk)
	p.mu.Unlock()
	return nil
}

func (p *blockingRecordingParser) snapshot() []StreamChunk {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]StreamChunk(nil), p.chunks...)
}
