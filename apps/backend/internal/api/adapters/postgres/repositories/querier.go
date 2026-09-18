// Package repositories contains the Postgres implementations of the storage
// ports. Each method joins the transaction carried by the context when present,
// or takes a connection from the pool otherwise (MANIFEST §5.3).
package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/txctx"
)

// querier is the subset of pgx used by the repositories. Both pgx.Tx and
// *pgxpool.Pool satisfy it, which is what allows the tx-or-pool switch.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// conn returns the transaction carried by ctx when present, or the pool
// otherwise.
func conn(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx := txctx.From(ctx); tx != nil {
		return tx
	}
	return pool
}
