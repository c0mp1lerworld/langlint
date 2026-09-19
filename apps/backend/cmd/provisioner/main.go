// Command provisioner runs the backend batch jobs (PRODUCT_DOMAIN §7.2):
//
//	go run ./cmd/provisioner refresh-aggregates
//	go run ./cmd/provisioner purge-raw-data
//	go run ./cmd/provisioner execute-deletions
//
// Run it from apps/backend. It requires APP_DATABASE_URL and applies the
// embedded migrations before running the selected job.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/fx"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/di"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/logger"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "provisioner:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("want exactly one subcommand, got %d (use %v)", len(args), handlers.CommandNames())
	}
	command := args[0]

	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}
	cfg, err := config.LoadProvisionerConfig()
	if err != nil {
		return err
	}

	log := logger.New(os.Stdout, slog.LevelInfo)

	var runner *handlers.Runner
	app := fx.New(
		fx.Supply(cfg, log),
		di.Module(),
		fx.Populate(&runner),
		fx.NopLogger,
	)
	if err := app.Err(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		return err
	}
	defer func() { _ = app.Stop(context.Background()) }()

	return runner.Run(ctx, command)
}
