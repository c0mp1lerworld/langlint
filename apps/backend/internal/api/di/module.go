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

	"github.com/c0mp1lerworld/langlint/backend/internal/api/accesslog"
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
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/db"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
	"github.com/c0mp1lerworld/langlint/backend/migrations"
)

// eventBusInterval is how often the outbox relay polls for unpublished events.
const eventBusInterval = time.Second

// Module is the Fx module of the HTTP API.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			newPool,
			newPseudonymizer,
			newPracticeRepository,
			newAnalysisRepository,
			newErrorMetricRepository,
			newProgressRepository,
			newDeletionRequestRepository,
			newAccessLogRepository,
			newStudySessionRepository,
			newQuizAttemptRepository,
			newUserEmail,
			newLLMTimeout,
			newUnitOfWork,
			newOutbox,
			newExtractor,
			newTutorQuestioner,
			newStudySessionGenerator,
			newDispatcher,
			newRelay,
		),
		fx.Provide(
			services.NewAnalysisService,
			services.NewPracticeService,
			services.NewAnalyticsService,
			services.NewIdentityService,
			services.NewQuizService,
			services.NewStudySessionService,
			newAnalysisRunner,
			event_handlers.NewAnalysisRequestedHandler,
			event_handlers.NewAnalysisCompletedHandler,
			event_handlers.NewAnalysisFailedHandler,
			event_handlers.NewWeaknessDetectedHandler,
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

func newErrorMetricRepository(pool *pgxpool.Pool, pseudonyms *pseudonymizer.Pseudonymizer) storage.ErrorMetricRepository {
	return repositories.NewErrorMetricRepository(pool, pseudonyms)
}

func newProgressRepository(pool *pgxpool.Pool) storage.ProgressRepository {
	return repositories.NewProgressRepository(pool)
}

// newPseudonymizer builds the HMAC keyed with APP_PSEUDONYM_SECRET (A8).
func newPseudonymizer(cfg config.ServerConfig) (*pseudonymizer.Pseudonymizer, error) {
	return pseudonymizer.New(cfg.PseudonymSecret)
}

func newDeletionRequestRepository(pool *pgxpool.Pool) storage.DeletionRequestRepository {
	return repositories.NewDeletionRequestRepository(pool)
}

func newAccessLogRepository(pool *pgxpool.Pool) storage.AccessLogRepository {
	return repositories.NewAccessLogRepository(pool)
}

func newStudySessionRepository(pool *pgxpool.Pool) storage.StudySessionRepository {
	return repositories.NewStudySessionRepository(pool)
}

func newQuizAttemptRepository(pool *pgxpool.Pool) storage.QuizAttemptRepository {
	return repositories.NewQuizAttemptRepository(pool)
}

// newUserEmail exposes the configured single-user email as an identity value.
func newUserEmail(cfg config.ServerConfig) identity.Email {
	return cfg.UserEmail
}

// newLLMTimeout exposes APP_LLM_TIMEOUT to the analysis service (4.3.3).
func newLLMTimeout(cfg config.ServerConfig) services.LLMTimeout {
	return services.LLMTimeout(cfg.LLMTimeout)
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

// newTutorQuestioner builds the active-practice engine over the same provider.
func newTutorQuestioner(cfg config.OpenAIConfig) ports.TutorQuestioner {
	options := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(llm.BaseURLOrDefault(cfg.BaseURL)),
	}
	return llm.NewOpenAITutorQuestioner(openai.NewClient(options...), cfg.Model)
}

// newStudySessionGenerator builds the study-session engine over the same provider.
func newStudySessionGenerator(cfg config.OpenAIConfig) ports.StudySessionGenerator {
	options := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(llm.BaseURLOrDefault(cfg.BaseURL)),
	}
	return llm.NewOpenAIStudySessionGenerator(openai.NewClient(options...), cfg.Model)
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
	weakness *event_handlers.WeaknessDetectedHandler,
) error {
	if err := dispatcher.Subscribe(domain.EventNameAnalysisRequested, requested); err != nil {
		return err
	}
	if err := dispatcher.Subscribe(domain.EventNameAnalysisCompleted, completed); err != nil {
		return err
	}
	if err := dispatcher.Subscribe(domain.EventNameAnalysisFailed, failed); err != nil {
		return err
	}
	return dispatcher.Subscribe(domain.EventNameWeaknessDetected, weakness)
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
func registerHTTP(lc fx.Lifecycle, cfg config.ServerConfig, logger *slog.Logger, server *httpapi.Server, accessLog storage.AccessLogRepository, shutdowner fx.Shutdowner) {
	router := httpx.NewRouter(logger)
	router.Use(httpx.CORS(cfg.CORSAllowedOrigins))
	router.Use(httpx.UserResolver(cfg.UserID))
	router.Use(accesslog.NewMiddleware(accessLog, logger).Wrap)
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
