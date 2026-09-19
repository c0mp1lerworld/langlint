//go:build integration

package e2e

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	provisionerdi "github.com/c0mp1lerworld/langlint/backend/internal/provisioner/di"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
)

func TestProvisionerDI_ModuleStartsAndStops(t *testing.T) {
	pool := startPool(t)

	cfg := config.ProvisionerConfig{
		DatabaseURL:   pool.Config().ConnString(),
		RawRetention:  30 * day,
		DeletionGrace: 30 * day,
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	var runner *handlers.Runner
	app := fx.New(
		fx.Supply(cfg, log),
		provisionerdi.Module(),
		fx.Populate(&runner),
		fx.NopLogger,
	)
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.Background()))
	require.NotNil(t, runner)
	require.NoError(t, app.Stop(context.Background()))
}
