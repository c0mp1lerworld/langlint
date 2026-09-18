package storage

import "context"

// UnitOfWork runs fn inside a single transaction (AP8). Services never open SQL
// transactions themselves; only the adapter touches pgx.
type UnitOfWork interface {
	InTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
