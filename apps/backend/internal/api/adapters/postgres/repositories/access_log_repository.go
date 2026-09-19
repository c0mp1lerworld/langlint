package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

// PostgresAccessLogRepository implements storage.AccessLogRepository. It is
// append-only: no method updates or deletes an event (A4, A9).
type PostgresAccessLogRepository struct {
	pool *pgxpool.Pool
}

var _ storage.AccessLogRepository = (*PostgresAccessLogRepository)(nil)

// NewAccessLogRepository builds a repository over the given pool.
func NewAccessLogRepository(pool *pgxpool.Pool) *PostgresAccessLogRepository {
	return &PostgresAccessLogRepository{pool: pool}
}

// Append stores an access event. An empty resource field is stored as NULL.
func (r *PostgresAccessLogRepository) Append(ctx context.Context, event *identity.AccessEvent) error {
	const q = `
INSERT INTO access_events (id, user_id, action, resource_type, resource_id, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		event.ID.String(),
		event.UserID.String(),
		event.Action,
		nullableText(event.ResourceType),
		nullableText(event.ResourceID),
		event.OccurredAt,
	); err != nil {
		return mapError("access_event", err)
	}
	return nil
}

// ListByUser returns a page of the user's events, newest first, plus the total.
func (r *PostgresAccessLogRepository) ListByUser(ctx context.Context, userID domain.ID, limit, offset int) ([]identity.AccessEvent, int, error) {
	q := conn(ctx, r.pool)

	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM access_events WHERE user_id = $1`, userID.String()).Scan(&total); err != nil {
		return nil, 0, mapError("access_event", err)
	}

	rows, err := q.Query(ctx,
		`SELECT id, user_id, action, resource_type, resource_id, occurred_at
		 FROM access_events
		 WHERE user_id = $1
		 ORDER BY occurred_at DESC, id
		 LIMIT $2 OFFSET $3`,
		userID.String(), limit, offset,
	)
	if err != nil {
		return nil, 0, mapError("access_event", err)
	}
	defer rows.Close()

	events := make([]identity.AccessEvent, 0)
	for rows.Next() {
		event, err := scanAccessEvent(rows)
		if err != nil {
			return nil, 0, mapError("access_event", err)
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapError("access_event", err)
	}

	return events, total, nil
}

func scanAccessEvent(row interface{ Scan(...any) error }) (*identity.AccessEvent, error) {
	var (
		id, userID               string
		action                   string
		resourceType, resourceID *string
		occurredAt               time.Time
	)
	if err := row.Scan(&id, &userID, &action, &resourceType, &resourceID, &occurredAt); err != nil {
		return nil, err
	}

	parsedID, err := domain.ParseID(id)
	if err != nil {
		return nil, fmt.Errorf("decode access event id: %w", err)
	}
	parsedUserID, err := domain.ParseID(userID)
	if err != nil {
		return nil, fmt.Errorf("decode access event user id: %w", err)
	}

	return &identity.AccessEvent{
		ID:           parsedID,
		UserID:       parsedUserID,
		Action:       action,
		ResourceType: derefText(resourceType),
		ResourceID:   derefText(resourceID),
		OccurredAt:   occurredAt,
	}, nil
}

// nullableText maps an empty string to a SQL NULL.
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// derefText maps a SQL NULL back to the empty string.
func derefText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
