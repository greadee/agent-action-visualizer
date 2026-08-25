package wrapper

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
)

const maxStreamChunkBytes = 32 << 10

type streamTap struct {
	ctx       context.Context
	cancel    context.CancelFunc
	parser    StructuredStreamParser
	emitter   adapter.Emitter
	queue     chan StreamChunk
	startOnce sync.Once
	startCh   chan struct{}
	done      chan struct{}
	sequence  atomic.Uint64
	finished  atomic.Bool
}

func newStreamTap(parent context.Context, parser StructuredStreamParser, emitter adapter.Emitter, queueSize int) *streamTap {
	ctx, cancel := context.WithCancel(parent)
	tap := &streamTap{
		ctx:     ctx,
		cancel:  cancel,
		parser:  parser,
		emitter: emitter,
		queue:   make(chan StreamChunk, queueSize),
		startCh: make(chan struct{}),
		done:    make(chan struct{}),
	}
	if parser == nil {
		close(tap.done)
		return tap
	}
	go tap.run()
	return tap
}

func (t *streamTap) start() {
	if t == nil || t.parser == nil {
		return
	}
	t.startOnce.Do(func() { close(t.startCh) })
}

func (t *streamTap) offer(stream Stream, data []byte) {
	if t == nil || t.parser == nil || t.finished.Load() {
		return
	}
	for len(data) > 0 {
		size := min(len(data), maxStreamChunkBytes)
		sequence := t.sequence.Add(1)
		chunk := StreamChunk{
			Stream:   stream,
			Data:     append([]byte(nil), data[:size]...),
			Sequence: sequence,
		}
		select {
		case t.queue <- chunk:
		default:
		}
		data = data[size:]
	}
}

func (t *streamTap) run() {
	defer close(t.done)
	select {
	case <-t.startCh:
	case <-t.ctx.Done():
		return
	}
	var last uint64
	for chunk := range t.queue {
		chunk.DroppedBefore = last != 0 && chunk.Sequence != last+1
		if last == 0 && chunk.Sequence > 1 {
			chunk.DroppedBefore = true
		}
		last = chunk.Sequence
		t.parse(chunk)
	}
	finalSequence := t.sequence.Load()
	dropped := finalSequence != last
	t.parse(StreamChunk{Stream: StreamStdout, Sequence: finalSequence + 1, DroppedBefore: dropped, EOF: true})
	t.parse(StreamChunk{Stream: StreamStderr, Sequence: finalSequence + 2, EOF: true})
}

func (t *streamTap) parse(chunk StreamChunk) {
	defer func() { _ = recover() }()
	events := t.parser.Parse(t.ctx, chunk)
	t.emitter.Emit(t.ctx, events)
}

func (t *streamTap) finish(deadline time.Duration) {
	if t == nil {
		return
	}
	if t.parser == nil {
		t.cancel()
		return
	}
	if t.finished.CompareAndSwap(false, true) {
		t.start()
		close(t.queue)
	}
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case <-t.done:
	case <-timer.C:
	}
	t.cancel()
}

type tapWriter struct {
	destination io.Writer
	stream      Stream
	tap         *streamTap
}

func observedWriter(destination io.Writer, stream Stream, tap *streamTap) io.Writer {
	if tap == nil || tap.parser == nil {
		return destination
	}
	return tapWriter{destination: destination, stream: stream, tap: tap}
}

func (w tapWriter) Write(data []byte) (int, error) {
	written, err := w.destination.Write(data)
	if written > 0 {
		w.tap.offer(w.stream, data[:written])
	}
	return written, err
}

var _ io.Writer = tapWriter{}
