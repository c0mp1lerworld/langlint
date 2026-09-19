//go:build integration

package e2e

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"go.uber.org/fx"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/di"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/testdb"
)

func TestDI_ModuleStartsAndStops(t *testing.T) {
	pool, cleanup, err := testdb.Start()
	if err != nil {
		t.Fatalf("testdb.Start() error = %v", err)
	}
	defer cleanup()

	serverCfg := config.ServerConfig{
		DatabaseURL: pool.Config().ConnString(),
		HTTPAddr:    "127.0.0.1:0",
		UserID:      domain.MustNewID(),
		LLMTimeout:  time.Second,
	}
	openaiCfg := config.OpenAIConfig{APIKey: "test-key", Model: "test-model", ModelVersion: "test-version"}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	app := fx.New(
		fx.Supply(serverCfg, openaiCfg, log),
		di.Module(),
		fx.NopLogger,
	)
	if err := app.Err(); err != nil {
		t.Fatalf("fx app failed to start: %v", err)
	}

	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("fx app failed to stop: %v", err)
	}
}
