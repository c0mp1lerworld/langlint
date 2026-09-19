package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
)

// PostgresErrorMetricRepository implements storage.ErrorMetricRepository. It
// materializes the user id pseudonymized, so the rebuilt metrics match the keys
// the API reads (A8).
type PostgresErrorMetricRepository struct {
	pool          *pgxpool.Pool
	pseudonymizer *pseudonymizer.Pseudonymizer
}

var _ storage.ErrorMetricRepository = (*PostgresErrorMetricRepository)(nil)

// NewErrorMetricRepository builds a repository over the given pool.
func NewErrorMetricRepository(pool *pgxpool.Pool, pseudonyms *pseudonymizer.Pseudonymizer) *PostgresErrorMetricRepository {
	return &PostgresErrorMetricRepository{pool: pool, pseudonymizer: pseudonyms}
}

// ReplaceAll wipes the materialized metrics and rewrites them from the source of
// truth in a single transaction, so a reader never observes a half-rebuilt
// table (A4).
func (r *PostgresErrorMetricRepository) ReplaceAll(ctx context.Context, metrics []*analytics.ErrorMetric) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapError("error_metric", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM error_metrics`); err != nil {
		return mapError("error_metric", err)
	}

	const insert = `
INSERT INTO error_metrics (user_id, code, "window", count, last_seen_at)
VALUES ($1, $2, $3, $4, $5)`

	if len(metrics) > 0 {
		batch := &pgx.Batch{}
		for _, m := range metrics {
			batch.Queue(insert, r.pseudonymizer.Pseudonymize(m.UserID.String()), string(m.Code), m.Window.String(), m.Count, m.LastSeenAt)
		}

		results := tx.SendBatch(ctx, batch)
		for range metrics {
			if _, err := results.Exec(); err != nil {
				_ = results.Close()
				return mapError("error_metric", err)
			}
		}
		if err := results.Close(); err != nil {
			return mapError("error_metric", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return mapError("error_metric", err)
	}
	return nil
}
