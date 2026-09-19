// Command api is the HTTP entry point of the backend (manifest §3.1). It loads
// the configuration, wires the Fx module and serves the OpenAPI contract.
//
// Run it from apps/backend:
//
//	go run ./cmd/api
package main

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/fx"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/di"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}

	serverCfg, err := config.LoadServerConfig()
	if err != nil {
		return err
	}
	openaiCfg, err := config.LoadOpenAIConfig()
	if err != nil {
		return err
	}

	log := logger.New(os.Stdout, slog.LevelInfo)

	app := fx.New(
		fx.Supply(serverCfg, openaiCfg, log),
		di.Module(),
		fx.NopLogger,
	)
	if err := app.Err(); err != nil {
		return err
	}

	app.Run()
	return nil
}
