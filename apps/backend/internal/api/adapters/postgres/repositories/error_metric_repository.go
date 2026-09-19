package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
)

// PostgresErrorMetricRepository implements storage.ErrorMetricRepository. It
// stores the user id pseudonymized (A8): analytics never materialize the raw
// identifier.
type PostgresErrorMetricRepository struct {
	pool          *pgxpool.Pool
	pseudonymizer *pseudonymizer.Pseudonymizer
}

var _ storage.ErrorMetricRepository = (*PostgresErrorMetricRepository)(nil)

// NewErrorMetricRepository builds a repository over the given pool. The
// pseudonymizer keys the stored user ids.
func NewErrorMetricRepository(pool *pgxpool.Pool, pseudonyms *pseudonymizer.Pseudonymizer) *PostgresErrorMetricRepository {
	return &PostgresErrorMetricRepository{pool: pool, pseudonymizer: pseudonyms}
}

// Upsert writes the metric by its natural key (pseudonymized user, code,
// window). It stores the absolute count from the aggregate, keeping repeated
// handler runs idempotent (AP7).
func (r *PostgresErrorMetricRepository) Upsert(ctx context.Context, m *analytics.ErrorMetric) error {
	const q = `
INSERT INTO error_metrics (user_id, code, "window", count, last_seen_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, code, "window") DO UPDATE SET
	count = EXCLUDED.count,
	last_seen_at = EXCLUDED.last_seen_at`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		r.pseudonymizer.Pseudonymize(m.UserID.String()),
		string(m.Code),
		m.Window.String(),
		m.Count,
		m.LastSeenAt,
	); err != nil {
		return mapError("error_metric", err)
	}
	return nil
}

// ListByUser returns the user's metrics for a given window. The user id is
// pseudonymized to match the stored key; the returned metrics carry the raw
// user id from the caller (the storage keying is an implementation detail).
func (r *PostgresErrorMetricRepository) ListByUser(ctx context.Context, userID domain.ID, window analytics.Window) ([]analytics.ErrorMetric, error) {
	rows, err := conn(ctx, r.pool).Query(ctx,
		`SELECT code, "window", count, last_seen_at
		 FROM error_metrics
		 WHERE user_id = $1 AND "window" = $2
		 ORDER BY count DESC, code`,
		r.pseudonymizer.Pseudonymize(userID.String()), window.String(),
	)
	if err != nil {
		return nil, mapError("error_metric", err)
	}
	defer rows.Close()

	metrics := make([]analytics.ErrorMetric, 0)
	for rows.Next() {
		m, err := scanErrorMetric(rows, userID)
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

func scanErrorMetric(row pgx.Row, userID domain.ID) (*analytics.ErrorMetric, error) {
	var (
		code, window string
		count        int
		lastSeenAt   time.Time
	)
	if err := row.Scan(&code, &window, &count, &lastSeenAt); err != nil {
		return nil, err
	}

	return &analytics.ErrorMetric{
		UserID:     userID,
		Code:       domain.ErrorPatternCode(code),
		Window:     analytics.Window(window),
		Count:      count,
		LastSeenAt: lastSeenAt,
	}, nil
}
