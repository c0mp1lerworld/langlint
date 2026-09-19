package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
)

// PostgresRawDataRepository implements storage.RawDataRepository.
type PostgresRawDataRepository struct {
	pool *pgxpool.Pool
}

var _ storage.RawDataRepository = (*PostgresRawDataRepository)(nil)

// NewRawDataRepository builds a repository over the given pool.
func NewRawDataRepository(pool *pgxpool.Pool) *PostgresRawDataRepository {
	return &PostgresRawDataRepository{pool: pool}
}

// PurgePracticesDeletedBefore hard-deletes the soft-deleted practices past the
// cutoff and their analyses in one transaction. Analyses are removed first to
// satisfy the foreign key. The transaction guarantees no study text survives
// while its analysis lingers (A8).
func (r *PostgresRawDataRepository) PurgePracticesDeletedBefore(ctx context.Context, cutoff time.Time) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, mapError("practice", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const deleteAnalyses = `
DELETE FROM analyses
WHERE practice_id IN (
	SELECT id FROM practices WHERE deleted_at IS NOT NULL AND deleted_at < $1
)`
	if _, err := tx.Exec(ctx, deleteAnalyses, cutoff); err != nil {
		return 0, mapError("practice", err)
	}

	tag, err := tx.Exec(ctx,
		`DELETE FROM practices WHERE deleted_at IS NOT NULL AND deleted_at < $1`,
		cutoff,
	)
	if err != nil {
		return 0, mapError("practice", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, mapError("practice", err)
	}
	return int(tag.RowsAffected()), nil
}
