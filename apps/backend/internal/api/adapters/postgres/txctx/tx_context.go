// Package txctx carries the active pgx transaction through context.Context
// inside the postgres adapter (AP8, MANIFEST §5.3). Keeping the key in a
// dedicated package lets both the unit of work and the repositories subpackage
// share it without exporting the key type.
package txctx

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

// With returns a copy of ctx carrying the transaction tx.
func With(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// From returns the transaction carried by ctx, or nil when there is none.
func From(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}
