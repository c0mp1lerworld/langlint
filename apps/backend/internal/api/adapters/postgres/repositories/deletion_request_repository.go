package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

// PostgresDeletionRequestRepository implements storage.DeletionRequestRepository.
type PostgresDeletionRequestRepository struct {
	pool *pgxpool.Pool
}

var _ storage.DeletionRequestRepository = (*PostgresDeletionRequestRepository)(nil)

// NewDeletionRequestRepository builds a repository over the given pool.
func NewDeletionRequestRepository(pool *pgxpool.Pool) *PostgresDeletionRequestRepository {
	return &PostgresDeletionRequestRepository{pool: pool}
}

// Append stores a new pending deletion request.
func (r *PostgresDeletionRequestRepository) Append(ctx context.Context, req *identity.DeletionRequest) error {
	const q = `
INSERT INTO deletion_requests (id, user_id, requested_at, executed_at)
VALUES ($1, $2, $3, $4)`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		req.ID.String(), req.UserID.String(), req.RequestedAt, req.ExecutedAt,
	); err != nil {
		return mapError("deletion_request", err)
	}
	return nil
}

// HasPending reports whether the user already has an unexecuted request.
func (r *PostgresDeletionRequestRepository) HasPending(ctx context.Context, userID domain.ID) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM deletion_requests WHERE user_id = $1 AND executed_at IS NULL)`

	var pending bool
	if err := conn(ctx, r.pool).QueryRow(ctx, q, userID.String()).Scan(&pending); err != nil {
		return false, mapError("deletion_request", err)
	}
	return pending, nil
}
