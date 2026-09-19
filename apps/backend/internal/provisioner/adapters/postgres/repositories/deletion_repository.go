package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
)

// PostgresDeletionRepository implements storage.DeletionRepository. Raw tables
// are keyed by the raw user id; analytics (error_metrics) are keyed by the
// pseudonym, so the purge must derive it too (A8/A9).
type PostgresDeletionRepository struct {
	pool          *pgxpool.Pool
	pseudonymizer *pseudonymizer.Pseudonymizer
}

var _ storage.DeletionRepository = (*PostgresDeletionRepository)(nil)

// NewDeletionRepository builds a repository over the given pool.
func NewDeletionRepository(pool *pgxpool.Pool, pseudonyms *pseudonymizer.Pseudonymizer) *PostgresDeletionRepository {
	return &PostgresDeletionRepository{pool: pool, pseudonymizer: pseudonyms}
}

// ListPending returns the deletion requests that have not been executed yet.
func (r *PostgresDeletionRepository) ListPending(ctx context.Context) ([]identity.DeletionRequest, error) {
	const q = `
SELECT id, user_id, requested_at, executed_at
FROM deletion_requests
WHERE executed_at IS NULL
ORDER BY requested_at, id`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, mapError("deletion_request", err)
	}
	defer rows.Close()

	requests := make([]identity.DeletionRequest, 0)
	for rows.Next() {
		var (
			id, userID  string
			requestedAt time.Time
			executedAt  *time.Time
		)
		if err := rows.Scan(&id, &userID, &requestedAt, &executedAt); err != nil {
			return nil, mapError("deletion_request", err)
		}

		parsedID, err := domain.ParseID(id)
		if err != nil {
			return nil, mapError("deletion_request", fmt.Errorf("decode id: %w", err))
		}
		parsedUserID, err := domain.ParseID(userID)
		if err != nil {
			return nil, mapError("deletion_request", fmt.Errorf("decode user id: %w", err))
		}

		requests = append(requests, identity.DeletionRequest{
			ID:          parsedID,
			UserID:      parsedUserID,
			RequestedAt: requestedAt,
			ExecutedAt:  executedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("deletion_request", err)
	}

	return requests, nil
}

// Execute hard-deletes every datum owned by the request's user and marks the
// request executed in one transaction, so the right to be forgotten is either
// fully materialized or not at all (A9).
func (r *PostgresDeletionRepository) Execute(ctx context.Context, req *identity.DeletionRequest) error {
	if req.ExecutedAt == nil {
		return &domain.InvalidStateError{Field: "executed_at", Message: "request must be marked executed before persisting"}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapError("deletion_request", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const deleteAnalyses = `
DELETE FROM analyses
WHERE practice_id IN (SELECT id FROM practices WHERE user_id = $1)`
	if _, err := tx.Exec(ctx, deleteAnalyses, req.UserID.String()); err != nil {
		return mapError("deletion_request", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM practices WHERE user_id = $1`, req.UserID.String()); err != nil {
		return mapError("deletion_request", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM error_metrics WHERE user_id = $1`, r.pseudonymizer.Pseudonymize(req.UserID.String())); err != nil {
		return mapError("deletion_request", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE deletion_requests SET executed_at = $2 WHERE id = $1 AND executed_at IS NULL`,
		req.ID.String(), *req.ExecutedAt,
	); err != nil {
		return mapError("deletion_request", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return mapError("deletion_request", err)
	}
	return nil
}
