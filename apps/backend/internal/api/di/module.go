// Package di wires the HTTP API entry point with Uber Fx (§2.1). It is the only
// place that instantiates concrete adapters and binds them to the ports the
// services depend on (A2).
package di

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.uber.org/fx"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/llm"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	httpapi "github.com/c0mp1lerworld/langlint/backend/internal/api/handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	portsevents "github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services/event_handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/db"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
	"github.com/c0mp1lerworld/langlint/backend/migrations"
)

// eventBusInterval is how often the outbox relay polls for unpublished events.
const eventBusInterval = time.Second

// Module is the Fx module of the HTTP API.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			newPool,
			newPracticeRepository,
			newAnalysisRepository,
			newErrorMetricRepository,
			newUnitOfWork,
			newOutbox,
			newExtractor,
			newDispatcher,
			newRelay,
		),
		fx.Provide(
			services.NewAnalysisService,
			services.NewPracticeService,
			services.NewAnalyticsService,
			newAnalysisRunner,
			event_handlers.NewAnalysisRequestedHandler,
			event_handlers.NewAnalysisCompletedHandler,
			event_handlers.NewAnalysisFailedHandler,
			httpapi.NewServer,
		),
		fx.Invoke(
			registerSubscriptions,
			runEventBus,
			registerHTTP,
		),
	)
}

// newPool opens the pgx pool, applies the embedded migrations and registers the
// pool lifecycle.
func newPool(lc fx.Lifecycle, cfg config.ServerConfig) (*pgxpool.Pool, error) {
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

func newPracticeRepository(pool *pgxpool.Pool) storage.PracticeRepository {
	return repositories.NewPracticeRepository(pool)
}

func newAnalysisRepository(pool *pgxpool.Pool) storage.AnalysisRepository {
	return repositories.NewAnalysisRepository(pool)
}

func newErrorMetricRepository(pool *pgxpool.Pool) storage.ErrorMetricRepository {
	return repositories.NewErrorMetricRepository(pool)
}

func newUnitOfWork(pool *pgxpool.Pool) storage.UnitOfWork {
	return postgres.NewUnitOfWork(pool)
}

func newOutbox(pool *pgxpool.Pool) portsevents.Outbox {
	return postgres.NewOutbox(pool)
}

func newExtractor(cfg config.OpenAIConfig) ports.LLMExtractor {
	options := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(llm.BaseURLOrDefault(cfg.BaseURL)),
	}
	return llm.NewOpenAIExtractor(openai.NewClient(options...), cfg.Model, cfg.ModelVersion)
}

func newDispatcher() *events.InMemoryEventDispatcher {
	return events.NewInMemoryEventDispatcher(events.DefaultBufferSize, events.DefaultWorkers)
}

func newRelay(pool *pgxpool.Pool, dispatcher *events.InMemoryEventDispatcher) *events.OutboxRelay {
	return events.NewOutboxRelay(pool, dispatcher, events.DefaultRelayBatch)
}

// newAnalysisRunner exposes the analysis service through the narrow interface
// the event handlers consume.
func newAnalysisRunner(s *services.AnalysisService) event_handlers.AnalysisRunner {
	return s
}

// registerSubscriptions connects the domain events to their handlers.
func registerSubscriptions(
	dispatcher *events.InMemoryEventDispatcher,
	requested *event_handlers.AnalysisRequestedHandler,
	completed *event_handlers.AnalysisCompletedHandler,
	failed *event_handlers.AnalysisFailedHandler,
) error {
	if err := dispatcher.Subscribe(domain.EventNameAnalysisRequested, requested); err != nil {
		return err
	}
	if err := dispatcher.Subscribe(domain.EventNameAnalysisCompleted, completed); err != nil {
		return err
	}
	return dispatcher.Subscribe(domain.EventNameAnalysisFailed, failed)
}

// runEventBus starts the dispatcher worker pool and the outbox relay with the
// application lifecycle.
func runEventBus(lc fx.Lifecycle, dispatcher *events.InMemoryEventDispatcher, relay *events.OutboxRelay) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			dispatcher.Run(ctx)
			go relay.Run(ctx, eventBusInterval)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

// registerHTTP mounts the generated router and runs the HTTP server with the
// application lifecycle.
func registerHTTP(lc fx.Lifecycle, cfg config.ServerConfig, logger *slog.Logger, server *httpapi.Server, shutdowner fx.Shutdowner) {
	router := httpx.NewRouter(logger)
	router.Use(httpx.UserResolver(cfg.UserID))
	httpapi.HandlerFromMux(server, router)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("http server stopped", "error", err)
					_ = shutdowner.Shutdown()
				}
			}()
			logger.Info("http server listening", "addr", cfg.HTTPAddr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return httpServer.Shutdown(ctx)
		},
	})
}
