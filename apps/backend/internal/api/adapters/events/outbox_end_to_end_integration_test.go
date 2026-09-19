//go:build integration

package events_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/testdb"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, cleanup, err := testdb.Start()
	if err != nil {
		panic(err)
	}
	pool = p
	code := m.Run()
	cleanup()
	os.Exit(code)
}

type recordingHandler struct {
	received chan domain.DomainEvent
}

func (h *recordingHandler) Handle(_ context.Context, event domain.DomainEvent) error {
	h.received <- event
	return nil
}

func resetOutbox(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE outbox_events")
	require.NoError(t, err)
}

func TestOutboxRelay_EndToEnd_DeliversCommittedEvent(t *testing.T) {
	resetOutbox(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := events.NewInMemoryEventDispatcher(16, 1)
	received := make(chan domain.DomainEvent, 1)
	require.NoError(t, dispatcher.Subscribe(domain.EventNamePracticeCreated, &recordingHandler{received: received}))
	dispatcher.Run(ctx)

	want := domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}
	uow := postgres.NewUnitOfWork(pool)
	outbox := postgres.NewOutbox(pool)
	require.NoError(t, uow.InTransaction(ctx, func(txCtx context.Context) error {
		return outbox.Append(txCtx, want)
	}))

	relay := events.NewOutboxRelay(pool, dispatcher, 100)
	require.NoError(t, relay.Tick(ctx))

	select {
	case got := <-received:
		pc, ok := got.(*domain.PracticeCreated)
		require.True(t, ok)
		require.Equal(t, want.PracticeID, pc.PracticeID)
		require.Equal(t, want.UserID, pc.UserID)
		require.Equal(t, want.Version, pc.Version)
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not receive the published event")
	}

	var publishedAt *time.Time
	require.NoError(t, pool.QueryRow(ctx, `SELECT published_at FROM outbox_events`).Scan(&publishedAt))
	require.NotNil(t, publishedAt, "event must be marked as published")
}

func TestOutboxRelay_EndToEnd_RollbackPublishesNothing(t *testing.T) {
	resetOutbox(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := events.NewInMemoryEventDispatcher(16, 1)
	received := make(chan domain.DomainEvent, 1)
	require.NoError(t, dispatcher.Subscribe(domain.EventNamePracticeCreated, &recordingHandler{received: received}))
	dispatcher.Run(ctx)

	boom := errors.New("boom")
	uow := postgres.NewUnitOfWork(pool)
	outbox := postgres.NewOutbox(pool)
	err := uow.InTransaction(ctx, func(txCtx context.Context) error {
		if appendErr := outbox.Append(txCtx, domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}); appendErr != nil {
			return appendErr
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	relay := events.NewOutboxRelay(pool, dispatcher, 100)
	require.NoError(t, relay.Tick(ctx))

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events`).Scan(&count))
	require.Zero(t, count, "rolled-back event must not exist")

	select {
	case <-received:
		t.Fatal("handler must not receive an event from a rolled-back transaction")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestOutboxRelay_Tick_UnknownEventType_IncrementsAttempts(t *testing.T) {
	resetOutbox(t)
	ctx := context.Background()
	id := domain.MustNewID()
	_, err := pool.Exec(ctx,
		`INSERT INTO outbox_events (id, event_type, payload, created_at, attempts) VALUES ($1, 'unknown.event', '{}'::jsonb, now(), 0)`,
		id.String(),
	)
	require.NoError(t, err)

	relay := events.NewOutboxRelay(pool, events.NewInMemoryEventDispatcher(4, 1), 100)
	require.NoError(t, relay.Tick(ctx))

	var (
		attempts    int
		publishedAt *time.Time
	)
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT attempts, published_at FROM outbox_events WHERE id = $1`, id.String(),
	).Scan(&attempts, &publishedAt))
	require.Equal(t, 1, attempts)
	require.Nil(t, publishedAt)
}

func TestOutboxRelay_Run_PublishesUntilCanceled(t *testing.T) {
	resetOutbox(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := events.NewInMemoryEventDispatcher(16, 1)
	received := make(chan domain.DomainEvent, 1)
	require.NoError(t, dispatcher.Subscribe(domain.EventNamePracticeCreated, &recordingHandler{received: received}))
	dispatcher.Run(ctx)

	require.NoError(t, postgres.NewOutbox(pool).Append(ctx, domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}))

	relay := events.NewOutboxRelay(pool, dispatcher, 100)
	go relay.Run(ctx, 20*time.Millisecond)

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not publish the pending event")
	}
}
