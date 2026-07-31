package wrapper

import (
	"context"
	"sync"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type delivery struct {
	collector adapter.Collector
	deadline  time.Duration
	queue     chan deliveryBatch
	stopOnce  sync.Once
	stopCh    chan struct{}
	done      chan struct{}
}

type deliveryBatch struct {
	events []protocol.Event
	ack    chan struct{}
}

func newDispatcher(collector adapter.Collector, queueSize int, deadline time.Duration) *delivery {
	value := &delivery{
		collector: collector,
		deadline:  deadline,
		queue:     make(chan deliveryBatch, queueSize),
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
	}
	go value.run()
	return value
}

func (d *delivery) Emit(ctx context.Context, events []protocol.Event) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	d.emit(events)
}

func (d *delivery) emit(events []protocol.Event) {
	d.offer(deliveryBatch{events: validEvents(events)})
}

func (d *delivery) emitFinal(events []protocol.Event) <-chan struct{} {
	ack := make(chan struct{})
	batch := deliveryBatch{events: validEvents(events), ack: ack}
	if d.offer(batch) {
		return ack
	}
	go func() {
		d.send(batch.events)
		close(ack)
	}()
	return ack
}

func (d *delivery) offer(batch deliveryBatch) bool {
	if d == nil || len(batch.events) == 0 {
		return false
	}
	select {
	case d.queue <- batch:
		return true
	default:
		return false
	}
}

func (d *delivery) run() {
	defer close(d.done)
	for {
		select {
		case batch := <-d.queue:
			d.send(batch.events)
			if batch.ack != nil {
				close(batch.ack)
			}
		case <-d.stopCh:
			return
		}
	}
}

func (d *delivery) send(events []protocol.Event) {
	if d.collector == nil || len(events) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), d.deadline)
	defer cancel()
	defer func() { _ = recover() }()
	_ = d.collector.Send(ctx, events)
}

func (d *delivery) flush(ack <-chan struct{}, timeout time.Duration) {
	if ack == nil {
		return
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ack:
	case <-timer.C:
	}
}

func (d *delivery) stop() {
	if d == nil {
		return
	}
	d.stopOnce.Do(func() { close(d.stopCh) })
	select {
	case <-d.done:
	case <-time.After(d.deadline):
	}
}

func validEvents(events []protocol.Event) []protocol.Event {
	if len(events) > adapter.DefaultMaxEvents {
		events = events[:adapter.DefaultMaxEvents]
	}
	valid := make([]protocol.Event, 0, len(events))
	for _, event := range events {
		if event.Validate() == nil {
			valid = append(valid, event)
		}
	}
	return valid
}
