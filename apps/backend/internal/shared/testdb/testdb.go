//go:build integration

// Package testdb boots an ephemeral Postgres 16 for Tier 3 tests and applies the
// embedded migrations. It lives in shared/ because both entry points (api and
// provisioner) need it and must not import each other (A2/A3). It is compiled
// only with -tags=integration (AP-MR8).
package testdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/c0mp1lerworld/langlint/backend/migrations"
)

// Start launches a postgres:16-alpine container and applies the migrations.
// The returned cleanup terminates the container and releases the pool.
func Start() (*pgxpool.Pool, func(), error) {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("langlint"),
		postgres.WithUsername("langlint"),
		postgres.WithPassword("langlint"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, err
	}
	terminate := func() { _ = container.Terminate(context.Background()) }

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate()
		return nil, nil, err
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		terminate()
		return nil, nil, err
	}

	if err := migrations.Up(pool); err != nil {
		pool.Close()
		terminate()
		return nil, nil, err
	}

	return pool, func() {
		pool.Close()
		terminate()
	}, nil
}
