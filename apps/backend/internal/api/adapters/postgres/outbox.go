package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	adapterevents "github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/txctx"
	portsevents "github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// PostgresOutbox persists domain events in the outbox_events table, joining the
// caller's transaction when present so data and event commit atomically (AP7,
// §5.1). It implements ports/events.Outbox.
type PostgresOutbox struct {
	pool *pgxpool.Pool
}

var _ portsevents.Outbox = (*PostgresOutbox)(nil)

// NewOutbox builds an outbox adapter over the given pool.
func NewOutbox(pool *pgxpool.Pool) *PostgresOutbox {
	return &PostgresOutbox{pool: pool}
}

// Append encodes the event and inserts it into the outbox within the current
// transaction (txctx), or in its own pool connection when there is none.
func (o *PostgresOutbox) Append(ctx context.Context, event domain.DomainEvent) error {
	eventType, payload, err := adapterevents.MarshalEvent(event)
	if err != nil {
		return &domain.InternalError{Field: "outbox", Message: "cannot encode event"}
	}
	id, err := domain.NewID()
	if err != nil {
		return &domain.InternalError{Field: "outbox", Message: "cannot generate event id"}
	}

	const q = `
INSERT INTO outbox_events (id, event_type, payload, created_at, published_at, attempts)
VALUES ($1, $2, $3::jsonb, $4, NULL, 0)`
	args := []any{id.String(), eventType, string(payload), time.Now().UTC()}

	if tx := txctx.From(ctx); tx != nil {
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return &domain.InternalError{Field: "outbox", Message: "database error"}
		}
		return nil
	}

	if _, err := o.pool.Exec(ctx, q, args...); err != nil {
		return &domain.InternalError{Field: "outbox", Message: "database error"}
	}
	return nil
}
