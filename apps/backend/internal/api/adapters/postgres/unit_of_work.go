// Package postgres contains the Postgres adapters for the API entry point.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/txctx"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// PostgresUnitOfWork is the only adapter allowed to open SQL transactions
// (AP8); it implements storage.UnitOfWork.
type PostgresUnitOfWork struct {
	pool *pgxpool.Pool
}

var _ storage.UnitOfWork = (*PostgresUnitOfWork)(nil)

// NewUnitOfWork builds a PostgresUnitOfWork over the given pool.
func NewUnitOfWork(pool *pgxpool.Pool) *PostgresUnitOfWork {
	return &PostgresUnitOfWork{pool: pool}
}

// InTransaction runs fn inside a transaction, committing on success and rolling
// back when fn returns an error. The transaction is exposed to repositories via
// the context (txctx).
func (u *PostgresUnitOfWork) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return &domain.InternalError{Field: "transaction", Message: "cannot begin transaction"}
	}

	if err := fn(txctx.With(ctx, tx)); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return &domain.InternalError{Field: "rollback", Message: "cannot rollback transaction"}
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return &domain.InternalError{Field: "commit", Message: "cannot commit transaction"}
	}
	return nil
}
