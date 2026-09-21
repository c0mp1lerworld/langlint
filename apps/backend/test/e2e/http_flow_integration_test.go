//go:build integration

// Package e2e exercises the HTTP API end to end against an ephemeral Postgres:
// create -> analyze -> poll -> completed -> analytics (AP-MR8).
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/accesslog"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	httpapi "github.com/c0mp1lerworld/langlint/backend/internal/api/handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services/event_handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/testdb"
)

// fakeExtractor returns a deterministic analysis without calling the LLM.
type fakeExtractor struct{}

func (fakeExtractor) Extract(_ context.Context, req ports.ExtractRequest) ([]analysis.Fragment, error) {
	return []analysis.Fragment{{
		SourceES:   req.SourceText,
		UserDraft:  req.DraftText,
		Correction: "corrected",
		TargetVerbReviews: []analysis.TargetVerbReview{{
			Verb:         "run",
			CorrectForm:  "ran",
			Rule:         "past simple",
			Why:          "acción pasada",
			ESContrast:   "en español varía",
			Alternatives: []string{"ran"},
		}},
		LexicalClarifications: []analysis.LexicalClarification{{
			Term:     "run",
			Meaning:  "correr",
			WhyWrong: "tiempo incorrecto",
		}},
		GrammarExplanations: []analysis.GrammarExplanation{{
			RuleName:       "past simple",
			Explanation:    "se usa para acciones terminadas",
			Construction:   "verbo + -ed / irregular",
			Counterexample: "run -> ran",
			Exception:      "verbos irregulares",
			ESContrast:     "en español el pretérito",
		}},
		ErrorPatterns: []domain.ErrorPattern{{
			Code:     domain.ErrorPatternCodeTenseAgreement,
			Severity: domain.ErrorPatternSeverityMinor,
			Note:     "tense",
		}},
	}}, nil
}

func (fakeExtractor) Model() string        { return "fake" }
func (fakeExtractor) ModelVersion() string { return "test" }

// fakeQuestioner returns a deterministic question/evaluation without the LLM.
type fakeQuestioner struct{}

func (fakeQuestioner) Question(_ context.Context, _ ports.QuestionRequest) (analysis.QuizQuestion, error) {
	return analysis.QuizQuestion{Kind: analysis.QuizKindFill, Prompt: "Completa: I bet ___ my team."}, nil
}

func (fakeQuestioner) Evaluate(_ context.Context, _ ports.EvaluateRequest) (analysis.QuizEvaluation, error) {
	return analysis.QuizEvaluation{Correct: true, Feedback: "¡Correcto!", FollowUp: "¿Por qué no 'in'?"}, nil
}

func newRouter(t *testing.T, pool *pgxpool.Pool, userID domain.ID) http.Handler {
	t.Helper()

	practices := repositories.NewPracticeRepository(pool)
	analyses := repositories.NewAnalysisRepository(pool)
	metrics := repositories.NewErrorMetricRepository(pool, testPseudonymizer(t))
	progress := repositories.NewProgressRepository(pool)
	deletions := repositories.NewDeletionRequestRepository(pool)
	accessLog := repositories.NewAccessLogRepository(pool)
	uow := postgres.NewUnitOfWork(pool)
	outbox := postgres.NewOutbox(pool)

	analysisSvc := services.NewAnalysisService(fakeExtractor{}, uow, practices, analyses, outbox, services.LLMTimeout(0))
	practiceSvc := services.NewPracticeService(uow, practices, analyses, outbox)
	analyticsSvc := services.NewAnalyticsService(metrics, progress)
	email, err := identity.NewEmail("student@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}
	identitySvc := services.NewIdentityService(email, practices, deletions, accessLog)

	dispatcher := events.NewInMemoryEventDispatcher(events.DefaultBufferSize, events.DefaultWorkers)
	relay := events.NewOutboxRelay(pool, dispatcher, events.DefaultRelayBatch)

	if err := dispatcher.Subscribe(domain.EventNameAnalysisRequested,
		event_handlers.NewAnalysisRequestedHandler(practices, analyses, analysisSvc)); err != nil {
		t.Fatalf("subscribe AnalysisRequested: %v", err)
	}
	if err := dispatcher.Subscribe(domain.EventNameAnalysisCompleted,
		event_handlers.NewAnalysisCompletedHandler(practices, metrics)); err != nil {
		t.Fatalf("subscribe AnalysisCompleted: %v", err)
	}
	if err := dispatcher.Subscribe(domain.EventNameAnalysisFailed,
		event_handlers.NewAnalysisFailedHandler(practices)); err != nil {
		t.Fatalf("subscribe AnalysisFailed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	dispatcher.Run(ctx)
	go relay.Run(ctx, 25*time.Millisecond)

	server := httpapi.NewServer(practiceSvc, analyticsSvc, identitySvc, services.NewQuizService(practices, analyses, fakeQuestioner{}))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpx.NewRouter(log)
	router.Use(httpx.UserResolver(userID))
	router.Use(accesslog.NewMiddleware(accessLog, log).Wrap)
	httpapi.HandlerFromMux(server, router)
	return router
}

func TestHTTPFlow_CreateAnalyzePollAnalytics(t *testing.T) {
	pool, cleanup, err := testdb.Start()
	if err != nil {
		t.Fatalf("testdb.Start() error = %v", err)
	}
	defer cleanup()

	userID := domain.MustNewID()
	api := httptest.NewServer(newRouter(t, pool, userID))
	defer api.Close()

	// [1] Create the practice.
	createBody := `{"source_text":"El perro escapó.","draft_text":"The dog escaped.","target_rules":[{"verb":"run","tense":"past simple"}]}`
	resp, err := http.Post(api.URL+"/practices", "application/json", bytes.NewBufferString(createBody))
	if err != nil {
		t.Fatalf("POST /practices error = %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /practices status = %d, want 201 (%s)", resp.StatusCode, body)
	}
	var created httpapi.Practice
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode practice: %v", err)
	}
	resp.Body.Close()

	// [2] Trigger the analysis.
	resp, err = http.Post(api.URL+"/practices/"+created.Id.String()+"/analyze", "application/json", nil)
	if err != nil {
		t.Fatalf("POST analyze error = %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST analyze status = %d, want 202 (%s)", resp.StatusCode, body)
	}
	resp.Body.Close()

	// [3] Poll until the practice is completed.
	deadline := time.Now().Add(10 * time.Second)
	var detail httpapi.PracticeDetail
	for time.Now().Before(deadline) {
		resp, err := http.Get(api.URL + "/practices/" + created.Id.String())
		if err != nil {
			t.Fatalf("GET practice error = %v", err)
		}
		err = json.NewDecoder(resp.Body).Decode(&detail)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("decode practice detail: %v", err)
		}
		if detail.Status != httpapi.PracticeStatus("analyzing") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if detail.Status != httpapi.PracticeStatus("completed") {
		t.Fatalf("practice status = %q, want completed", detail.Status)
	}
	if detail.Analysis == nil || len(detail.Analysis.Fragments) != 1 {
		t.Fatalf("analysis = %+v, want one fragment", detail.Analysis)
	}

	// [4] The analytics endpoint reflects the completed analysis.
	resp, err = http.Get(api.URL + "/analytics/error-patterns?window=week")
	if err != nil {
		t.Fatalf("GET analytics error = %v", err)
	}
	var stats httpapi.ErrorPatternStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	resp.Body.Close()

	if len(stats.Patterns) != 1 || stats.Patterns[0].Count != 1 {
		t.Fatalf("patterns = %+v, want one pattern with count 1", stats.Patterns)
	}
	if stats.Patterns[0].Code != httpapi.ErrorPatternCode(domain.ErrorPatternCodeTenseAgreement) {
		t.Fatalf("pattern code = %q", stats.Patterns[0].Code)
	}

	// [4b] The progress endpoint returns the derived series (was 501, GAP-1).
	resp, err = http.Get(api.URL + "/analytics/progress?window=week")
	if err != nil {
		t.Fatalf("GET progress error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET progress status = %d, want 200 (%s)", resp.StatusCode, body)
	}
	var series httpapi.ProgressSeries
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		t.Fatalf("decode progress series: %v", err)
	}
	resp.Body.Close()
	if series.Window != httpapi.Window(analytics.WindowWeek) || len(series.Points) != 1 {
		t.Fatalf("progress series = %+v, want one weekly point", series)
	}
	if series.Points[0].TotalFragments != 1 || series.Points[0].ErrorCount != 1 {
		t.Fatalf("progress point = %+v, want total=1 errors=1", series.Points[0])
	}

	// [5] The active-practice quiz generates a question and evaluates an answer.
	resp, err = http.Post(api.URL+"/practices/"+created.Id.String()+"/quiz", "application/json",
		bytes.NewBufferString(`{"fragment_index":0}`))
	if err != nil {
		t.Fatalf("POST quiz error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST quiz status = %d, want 200 (%s)", resp.StatusCode, body)
	}
	var question httpapi.QuizQuestion
	if err := json.NewDecoder(resp.Body).Decode(&question); err != nil {
		t.Fatalf("decode quiz question: %v", err)
	}
	resp.Body.Close()
	if question.Prompt == "" {
		t.Fatal("quiz question prompt is empty")
	}

	resp, err = http.Post(api.URL+"/practices/"+created.Id.String()+"/quiz/answer", "application/json",
		bytes.NewBufferString(`{"fragment_index":0,"question":"I bet ___ my team.","answer":"on"}`))
	if err != nil {
		t.Fatalf("POST quiz answer error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST quiz answer status = %d, want 200 (%s)", resp.StatusCode, body)
	}
	var evaluation httpapi.QuizEvaluation
	if err := json.NewDecoder(resp.Body).Decode(&evaluation); err != nil {
		t.Fatalf("decode quiz evaluation: %v", err)
	}
	resp.Body.Close()
	if !evaluation.Correct || evaluation.Feedback == "" {
		t.Fatalf("quiz evaluation = %+v", evaluation)
	}
}
