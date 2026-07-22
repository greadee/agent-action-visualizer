package ingest

import (
	"context"
	"sync"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type Handler func(context.Context, protocol.Event)

type Collector struct {
	queue   *Queue
	wake    chan struct{}
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
	handler Handler
}

func NewCollector(capacity int, handler Handler) *Collector {
	return &Collector{queue: NewQueue(capacity), wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}), handler: handler}
}

func (c *Collector) Start(ctx context.Context) { go c.run(ctx) }
func (c *Collector) Submit(event protocol.Event) bool {
	accepted := c.queue.Offer(event)
	if accepted {
		select {
		case c.wake <- struct{}{}:
		default:
		}
	}
	return accepted
}
func (c *Collector) Stop()             { c.once.Do(func() { close(c.stop); <-c.done }) }
func (c *Collector) Stats() QueueStats { return c.queue.Stats() }

func (c *Collector) run(ctx context.Context) {
	defer close(c.done)
	for {
		for {
			event, ok := c.queue.Pop()
			if !ok {
				break
			}
			if c.handler != nil {
				c.handler(ctx, event)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-c.stop:
			return
		case <-c.wake:
		case <-time.After(250 * time.Millisecond):
		}
	}
}
