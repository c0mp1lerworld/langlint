package events

import (
	"context"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

type handlerFunc func(ctx context.Context, event domain.DomainEvent) error

func (f handlerFunc) Handle(ctx context.Context, event domain.DomainEvent) error {
	return f(ctx, event)
}

func TestInMemoryEventDispatcher_Subscribe_DeliversEvent(t *testing.T) {
	dispatcher := NewInMemoryEventDispatcher(16, 1)
	received := make(chan domain.DomainEvent, 1)
	if err := dispatcher.Subscribe(domain.EventNamePracticeCreated, handlerFunc(func(_ context.Context, event domain.DomainEvent) error {
		received <- event
		return nil
	})); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dispatcher.Run(ctx)

	event := domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}
	if err := dispatcher.Dispatch(ctx, event); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	select {
	case got := <-received:
		if got.EventName() != domain.EventNamePracticeCreated {
			t.Fatalf("delivered %q, want %q", got.EventName(), domain.EventNamePracticeCreated)
		}
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}
}

func TestInMemoryEventDispatcher_Dispatch_NoSubscribers_NoError(t *testing.T) {
	dispatcher := NewInMemoryEventDispatcher(4, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dispatcher.Run(ctx)

	if err := dispatcher.Dispatch(ctx, domain.PracticeCreated{Version: 1}); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
}

func TestInMemoryEventDispatcher_Dispatch_BufferFull_DropsEvent(t *testing.T) {
	dispatcher := NewInMemoryEventDispatcher(1, 1)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := dispatcher.Dispatch(ctx, domain.PracticeCreated{Version: 1}); err != nil {
			t.Fatalf("Dispatch() error = %v", err)
		}
	}

	if got := len(dispatcher.ch); got != 1 {
		t.Fatalf("buffer length = %d, want 1 (drop policy)", got)
	}
}

func TestNewInMemoryEventDispatcher_NonPositive_FallsBackToDefaults(t *testing.T) {
	dispatcher := NewInMemoryEventDispatcher(0, 0)

	if cap(dispatcher.ch) != DefaultBufferSize {
		t.Fatalf("buffer capacity = %d, want %d", cap(dispatcher.ch), DefaultBufferSize)
	}
	if dispatcher.workers != DefaultWorkers {
		t.Fatalf("workers = %d, want %d", dispatcher.workers, DefaultWorkers)
	}
}
