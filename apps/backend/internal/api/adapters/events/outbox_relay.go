package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	portsevents "github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// DefaultRelayBatch is the maximum number of events published per tick.
const DefaultRelayBatch = 100

// OutboxRelay reads unpublished events from the outbox and hands them to the
// dispatcher post-commit (AP7, §5.1). It runs in the same process as the
// dispatcher; cross-binary delivery goes through the outbox table.
type OutboxRelay struct {
	pool       *pgxpool.Pool
	dispatcher portsevents.EventDispatcher
	batch      int
}

// NewOutboxRelay builds a relay. A non-positive batch falls back to
// DefaultRelayBatch.
func NewOutboxRelay(pool *pgxpool.Pool, dispatcher portsevents.EventDispatcher, batch int) *OutboxRelay {
	if batch <= 0 {
		batch = DefaultRelayBatch
	}
	return &OutboxRelay{pool: pool, dispatcher: dispatcher, batch: batch}
}

// Run ticks the relay every interval until ctx is canceled.
func (r *OutboxRelay) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Tick(ctx); err != nil {
				slog.ErrorContext(ctx, "outbox relay tick failed", "error", err)
			}
		}
	}
}

// Tick publishes one batch of pending events and marks them as published.
// Events that cannot be decoded or dispatched get their attempts incremented
// and stay pending for a later tick.
func (r *OutboxRelay) Tick(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return &domain.InternalError{Field: "relay", Message: "cannot begin transaction"}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, event_type, payload, created_at, attempts
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at, id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, r.batch)
	if err != nil {
		return &domain.InternalError{Field: "relay", Message: "database error"}
	}

	pending, err := scanPending(rows)
	rows.Close()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, event := range pending {
		if err := r.publish(ctx, tx, event, now); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return &domain.InternalError{Field: "relay", Message: "cannot commit transaction"}
	}
	return nil
}

func (r *OutboxRelay) publish(ctx context.Context, tx pgx.Tx, pending domain.OutboxEvent, now time.Time) error {
	event, err := UnmarshalEvent(pending.EventType, pending.Payload)
	if err != nil {
		slog.ErrorContext(ctx, "outbox relay cannot decode event", "event_id", pending.ID.String(), "event_type", pending.EventType)
		return bumpAttempts(ctx, tx, pending.ID)
	}

	if err := r.dispatcher.Dispatch(ctx, event); err != nil {
		slog.ErrorContext(ctx, "outbox relay cannot dispatch event", "event_id", pending.ID.String(), "event_type", pending.EventType)
		return bumpAttempts(ctx, tx, pending.ID)
	}

	if _, err := tx.Exec(ctx, `UPDATE outbox_events SET published_at = $2 WHERE id = $1`, pending.ID.String(), now); err != nil {
		return &domain.InternalError{Field: "relay", Message: "database error"}
	}
	return nil
}

func scanPending(rows pgx.Rows) ([]domain.OutboxEvent, error) {
	pending := make([]domain.OutboxEvent, 0)
	for rows.Next() {
		var (
			id     string
			record domain.OutboxEvent
		)
		if err := rows.Scan(&id, &record.EventType, &record.Payload, &record.CreatedAt, &record.Attempts); err != nil {
			return nil, &domain.InternalError{Field: "relay", Message: "database error"}
		}
		parsed, err := domain.ParseID(id)
		if err != nil {
			return nil, &domain.InternalError{Field: "relay", Message: "database error"}
		}
		record.ID = parsed
		pending = append(pending, record)
	}
	if err := rows.Err(); err != nil {
		return nil, &domain.InternalError{Field: "relay", Message: "database error"}
	}
	return pending, nil
}

func bumpAttempts(ctx context.Context, tx pgx.Tx, id domain.ID) error {
	if _, err := tx.Exec(ctx, `UPDATE outbox_events SET attempts = attempts + 1 WHERE id = $1`, id.String()); err != nil {
		return &domain.InternalError{Field: "relay", Message: "database error"}
	}
	return nil
}
