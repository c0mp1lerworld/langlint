// Command migrate applies the embedded schema migrations (goose) to the
// database configured through APP_DATABASE_URL. It is idempotent: goose tracks
// the applied versions. Run it from apps/backend:
//
//	go run ./cmd/migrate
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/db"
	"github.com/c0mp1lerworld/langlint/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}

	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrations.Up(pool); err != nil {
		return err
	}

	fmt.Println("migrate: schema up to date")
	return nil
}
