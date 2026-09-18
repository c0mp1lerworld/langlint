package events

import (
	"context"
	"sync"

	portsevents "github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Default buffer size and worker pool of the in-memory bus (§5.1).
const (
	DefaultBufferSize = 1024
	DefaultWorkers    = 4
)

// InMemoryEventDispatcher delivers domain events to subscribed handlers in the
// same process. It uses a buffered channel with a drop policy for backpressure
// and a worker pool to drain it (§5.1).
type InMemoryEventDispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]portsevents.EventHandler
	ch       chan domain.DomainEvent
	workers  int
}

var _ portsevents.EventDispatcher = (*InMemoryEventDispatcher)(nil)

// NewInMemoryEventDispatcher builds a dispatcher. Non-positive bufferSize or
// workers fall back to the defaults.
func NewInMemoryEventDispatcher(bufferSize, workers int) *InMemoryEventDispatcher {
	if bufferSize <= 0 {
		bufferSize = DefaultBufferSize
	}
	if workers <= 0 {
		workers = DefaultWorkers
	}
	return &InMemoryEventDispatcher{
		handlers: make(map[string][]portsevents.EventHandler),
		ch:       make(chan domain.DomainEvent, bufferSize),
		workers:  workers,
	}
}

// Subscribe registers a handler for the given event name.
func (d *InMemoryEventDispatcher) Subscribe(eventName string, handler portsevents.EventHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventName] = append(d.handlers[eventName], handler)
	return nil
}

// Dispatch enqueues the event. When the buffer is full the event is dropped
// (backpressure policy) instead of blocking the relay; services never call this
// directly, they use the outbox (AP7).
func (d *InMemoryEventDispatcher) Dispatch(ctx context.Context, event domain.DomainEvent) error {
	select {
	case d.ch <- event:
	default:
	}
	return nil
}

// Run starts the worker pool that drains the channel until ctx is canceled.
func (d *InMemoryEventDispatcher) Run(ctx context.Context) {
	for i := 0; i < d.workers; i++ {
		go d.worker(ctx)
	}
}

func (d *InMemoryEventDispatcher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-d.ch:
			d.deliver(ctx, event)
		}
	}
}

func (d *InMemoryEventDispatcher) deliver(ctx context.Context, event domain.DomainEvent) {
	d.mu.RLock()
	handlers := d.handlers[event.EventName()]
	d.mu.RUnlock()

	for _, handler := range handlers {
		_ = handler.Handle(ctx, event)
	}
}
