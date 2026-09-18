package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// PostgresErrorMetricRepository implements storage.ErrorMetricRepository.
type PostgresErrorMetricRepository struct {
	pool *pgxpool.Pool
}

var _ storage.ErrorMetricRepository = (*PostgresErrorMetricRepository)(nil)

// NewErrorMetricRepository builds a repository over the given pool.
func NewErrorMetricRepository(pool *pgxpool.Pool) *PostgresErrorMetricRepository {
	return &PostgresErrorMetricRepository{pool: pool}
}

// Upsert writes the metric by its natural key (user, code, window). It stores
// the absolute count from the aggregate, keeping repeated handler runs
// idempotent (AP7).
func (r *PostgresErrorMetricRepository) Upsert(ctx context.Context, m *analytics.ErrorMetric) error {
	const q = `
INSERT INTO error_metrics (user_id, code, "window", count, last_seen_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, code, "window") DO UPDATE SET
	count = EXCLUDED.count,
	last_seen_at = EXCLUDED.last_seen_at`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		m.UserID.String(),
		string(m.Code),
		m.Window.String(),
		m.Count,
		m.LastSeenAt,
	); err != nil {
		return mapError("error_metric", err)
	}
	return nil
}

// ListByUser returns the user's metrics for a given window.
func (r *PostgresErrorMetricRepository) ListByUser(ctx context.Context, userID domain.ID, window analytics.Window) ([]analytics.ErrorMetric, error) {
	rows, err := conn(ctx, r.pool).Query(ctx,
		`SELECT user_id, code, "window", count, last_seen_at
		 FROM error_metrics
		 WHERE user_id = $1 AND "window" = $2
		 ORDER BY count DESC, code`,
		userID.String(), window.String(),
	)
	if err != nil {
		return nil, mapError("error_metric", err)
	}
	defer rows.Close()

	metrics := make([]analytics.ErrorMetric, 0)
	for rows.Next() {
		m, err := scanErrorMetric(rows)
		if err != nil {
			return nil, mapError("error_metric", err)
		}
		metrics = append(metrics, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("error_metric", err)
	}

	return metrics, nil
}

func scanErrorMetric(row pgx.Row) (*analytics.ErrorMetric, error) {
	var (
		userID, code, window string
		count                int
		lastSeenAt           time.Time
	)
	if err := row.Scan(&userID, &code, &window, &count, &lastSeenAt); err != nil {
		return nil, err
	}

	parsedUserID, err := domain.ParseID(userID)
	if err != nil {
		return nil, fmt.Errorf("decode error metric user id: %w", err)
	}

	return &analytics.ErrorMetric{
		UserID:     parsedUserID,
		Code:       domain.ErrorPatternCode(code),
		Window:     analytics.Window(window),
		Count:      count,
		LastSeenAt: lastSeenAt,
	}, nil
}
