// Package di wires the provisioner entry point with Uber Fx (§2.1). It is the
// only place that instantiates the provisioner adapters and binds them to the
// ports its services depend on (A2). It shares no module with the api entry
// point: api and provisioner are independent (manifest §3.3).
package di

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/db"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
	"github.com/c0mp1lerworld/langlint/backend/migrations"
)

// Module is the Fx module of the provisioner batch entry point.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			newPool,
			newPseudonymizer,
			newAnalyticsSourceRepository,
			newErrorMetricRepository,
			newRawDataRepository,
			newDeletionRepository,
			services.NewRefreshAggregatesService,
			newPurgeRawDataService,
			newExecuteDeletionsService,
			handlers.NewRunner,
		),
	)
}

// newPool opens the pgx pool, applies the embedded migrations and registers the
// pool lifecycle. Applying migrations here is idempotent (goose) and keeps the
// batch jobs self-sufficient.
func newPool(lc fx.Lifecycle, cfg config.ProvisionerConfig) (*pgxpool.Pool, error) {
	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := migrations.Up(pool); err != nil {
		pool.Close()
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		pool.Close()
		return nil
	}})
	return pool, nil
}

func newAnalyticsSourceRepository(pool *pgxpool.Pool) storage.AnalyticsSourceRepository {
	return repositories.NewAnalyticsSourceRepository(pool)
}

func newErrorMetricRepository(pool *pgxpool.Pool, pseudonyms *pseudonymizer.Pseudonymizer) storage.ErrorMetricRepository {
	return repositories.NewErrorMetricRepository(pool, pseudonyms)
}

func newRawDataRepository(pool *pgxpool.Pool) storage.RawDataRepository {
	return repositories.NewRawDataRepository(pool)
}

func newDeletionRepository(pool *pgxpool.Pool, pseudonyms *pseudonymizer.Pseudonymizer) storage.DeletionRepository {
	return repositories.NewDeletionRepository(pool, pseudonyms)
}

// newPseudonymizer builds the HMAC keyed with APP_PSEUDONYM_SECRET (A8).
func newPseudonymizer(cfg config.ProvisionerConfig) (*pseudonymizer.Pseudonymizer, error) {
	return pseudonymizer.New(cfg.PseudonymSecret)
}

func newPurgeRawDataService(raw storage.RawDataRepository, cfg config.ProvisionerConfig) *services.PurgeRawDataService {
	return services.NewPurgeRawDataService(raw, cfg.RawRetention)
}

func newExecuteDeletionsService(deletions storage.DeletionRepository, cfg config.ProvisionerConfig) *services.ExecuteDeletionsService {
	return services.NewExecuteDeletionsService(deletions, cfg.DeletionGrace)
}
