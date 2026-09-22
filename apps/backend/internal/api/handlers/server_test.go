package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
)

func runUoW(ctrl *gomock.Controller) *mocks.MockUnitOfWork {
	uow := mocks.NewMockUnitOfWork(ctrl)
	uow.EXPECT().InTransaction(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	).AnyTimes()
	return uow
}

// wireID converts a domain id into the generated path-parameter type.
func wireID(id domain.ID) PracticeId {
	return PracticeId(id)
}

func newTestServer(ctrl *gomock.Controller, practices *mocks.MockPracticeRepository, analyses *mocks.MockAnalysisRepository, metrics *mocks.MockErrorMetricRepository, outbox *mocks.MockOutbox) *Server {
	practiceSvc := services.NewPracticeService(runUoW(ctrl), practices, analyses, outbox)
	return NewServer(practiceSvc, services.NewAnalyticsService(metrics, mocks.NewMockProgressRepository(ctrl)), testIdentityService(ctrl), testQuizService(ctrl), testTutorService(ctrl))
}

// testQuizService builds a QuizService over throwaway mocks.
func testQuizService(ctrl *gomock.Controller) *services.QuizService {
	return services.NewQuizService(
		mocks.NewMockPracticeRepository(ctrl),
		mocks.NewMockAnalysisRepository(ctrl),
		mocks.NewMockTutorQuestioner(ctrl),
	)
}

// testTutorService builds a StudySessionService over throwaway mocks.
func testTutorService(ctrl *gomock.Controller) *services.StudySessionService {
	return services.NewStudySessionService(
		mocks.NewMockErrorMetricRepository(ctrl),
		mocks.NewMockStudySessionRepository(ctrl),
		mocks.NewMockStudySessionGenerator(ctrl),
	)
}

// testEmail is the configured single-user email used by the handler tests.
const testEmail = "student@example.com"

func mustEmail(t *testing.T, raw string) identity.Email {
	t.Helper()
	email, err := identity.NewEmail(raw)
	if err != nil {
		t.Fatalf("NewEmail(%q) error = %v", raw, err)
	}
	return email
}

// testIdentityService builds an IdentityService over throwaway mocks.
func testIdentityService(ctrl *gomock.Controller) *services.IdentityService {
	email, err := identity.NewEmail(testEmail)
	if err != nil {
		panic(err)
	}
	return services.NewIdentityService(
		email,
		mocks.NewMockPracticeRepository(ctrl),
		mocks.NewMockDeletionRequestRepository(ctrl),
		mocks.NewMockAccessLogRepository(ctrl),
	)
}

// serve runs h with the user injected, mimicking the router middleware chain.
func serve(userID domain.ID, h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	httpx.UserResolver(userID)(h).ServeHTTP(rec, req)
	return rec
}

func draftPractice(userID domain.ID) *practice.Practice {
	return &practice.Practice{
		ID:          domain.MustNewID(),
		UserID:      userID,
		SourceText:  practice.SourceText("El perro escapó."),
		DraftText:   practice.DraftText("The dog escaped."),
		TargetRules: []practice.TargetRule{{Verb: "run", Tense: "past simple", Note: "irregular"}},
		Status:      practice.PracticeStatusDraft,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
}

func TestServer_CreatePractice_Valid_Returns201(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	outbox := mocks.NewMockOutbox(ctrl)
	outbox.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), outbox)
	req := httptest.NewRequest(http.MethodPost, "/practices",
		strings.NewReader(`{"source_text":"El perro escapó.","draft_text":"The dog escaped.","target_rules":[{"verb":"run","tense":"past simple"}]}`))

	rec := serve(userID, srv.CreatePractice, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}

	var body Practice
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Status != PracticeStatus(practice.PracticeStatusDraft) {
		t.Fatalf("status = %q, want draft", body.Status)
	}
	if body.SourceText != "El perro escapó." || body.UserId.String() != userID.String() {
		t.Fatalf("body = %+v", body)
	}
}

func TestServer_CreatePractice_MalformedJSON_Returns422(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv := newTestServer(ctrl, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodPost, "/practices", strings.NewReader(`{`))

	rec := serve(domain.MustNewID(), srv.CreatePractice, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestServer_CreatePractice_EmptyRules_Returns422(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv := newTestServer(ctrl, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodPost, "/practices",
		strings.NewReader(`{"source_text":"x","draft_text":"y","target_rules":[]}`))

	rec := serve(domain.MustNewID(), srv.CreatePractice, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestServer_MissingUser_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv := newTestServer(ctrl, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodPost, "/practices", strings.NewReader(`{}`))

	rec := httptest.NewRecorder()
	srv.CreatePractice(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestServer_GetPractice_WithAnalysis_Returns200(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)
	a := &analysis.Analysis{
		ID:           domain.MustNewID(),
		PracticeID:   p.ID,
		Model:        "gpt-4o-mini",
		ModelVersion: "2024-07-18",
		Status:       analysis.AnalysisStatusCompleted,
		Fragments: []analysis.Fragment{{
			SourceES:      "El perro escapó.",
			UserDraft:     "The dog escaped.",
			Correction:    "The dog has escaped.",
			ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeTenseAgreement, Severity: domain.ErrorPatternSeverityMinor, Note: "tense"}},
		}},
	}

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(a, nil)

	srv := newTestServer(ctrl, practices, analyses, mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/practices/"+p.ID.String(), nil)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) { srv.GetPractice(w, r, wireID(p.ID)) }, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body PracticeDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Analysis == nil || len(body.Analysis.Fragments) != 1 {
		t.Fatalf("analysis = %+v, want one fragment", body.Analysis)
	}
	if body.Analysis.Fragments[0].SourceEs != "El perro escapó." {
		t.Fatalf("fragment = %+v", body.Analysis.Fragments[0])
	}
}

func TestServer_ListPractices_ReturnsPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().ListByUser(gomock.Any(), userID, 20, 0).Return([]practice.Practice{*p}, 1, nil)

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/practices", nil)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) {
		srv.ListPractices(w, r, ListPracticesParams{})
	}, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body PracticeList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Total != 1 || len(body.Items) != 1 {
		t.Fatalf("body = %+v, want one item", body)
	}
}

func TestServer_AnalyzePractice_Returns202WithRetryAfter(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	outbox := mocks.NewMockOutbox(ctrl)
	outbox.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), outbox)
	req := httptest.NewRequest(http.MethodPost, "/practices/"+p.ID.String()+"/analyze", nil)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) { srv.AnalyzePractice(w, r, wireID(p.ID)) }, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != retryAfterSeconds {
		t.Fatalf("Retry-After = %q, want %q", got, retryAfterSeconds)
	}
}

func TestServer_DeletePractice_Returns204(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodDelete, "/practices/"+p.ID.String(), nil)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) { srv.DeletePractice(w, r, wireID(p.ID)) }, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestServer_GetErrorPatternStats_ReturnsPatterns(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, analytics.WindowWeek).Return([]analytics.ErrorMetric{{
		UserID:     userID,
		Code:       domain.ErrorPatternCodeTenseAgreement,
		Window:     analytics.WindowWeek,
		Count:      4,
		LastSeenAt: time.Unix(0, 0).UTC(),
	}}, nil)

	srv := newTestServer(ctrl, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), metrics, mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/analytics/error-patterns", nil)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) {
		srv.GetErrorPatternStats(w, r, GetErrorPatternStatsParams{})
	}, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body ErrorPatternStats
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Window != Window(analytics.WindowWeek) || len(body.Patterns) != 1 || body.Patterns[0].Count != 4 {
		t.Fatalf("body = %+v", body)
	}
}

func TestServer_ProgressSeries_ReturnsSeries(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	samples := []analytics.ProgressSample{
		{CompletedAt: time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC), TotalFragments: 4, ErrorCount: 1},
	}

	progress := mocks.NewMockProgressRepository(ctrl)
	progress.EXPECT().ListSamplesByUser(gomock.Any(), userID).Return(samples, nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), progress),
		testIdentityService(ctrl),
		testQuizService(ctrl),
		testTutorService(ctrl),
	)

	rec := serve(userID,
		func(w http.ResponseWriter, r *http.Request) { srv.GetProgressSeries(w, r, GetProgressSeriesParams{}) },
		httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body ProgressSeries
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Window != Window(analytics.WindowWeek) || len(body.Points) != 1 {
		t.Fatalf("body = %+v", body)
	}
	if body.Points[0].TotalFragments != 4 || body.Points[0].ErrorCount != 1 {
		t.Fatalf("point = %+v, want total=4 errors=1", body.Points[0])
	}
}

func TestServer_ProgressSeries_InvalidWindow_Returns422(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	srv := newTestServer(ctrl, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))

	invalid := Window("year")
	rec := serve(userID,
		func(w http.ResponseWriter, r *http.Request) {
			srv.GetProgressSeries(w, r, GetProgressSeriesParams{Window: &invalid})
		},
		httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestServer_ExportData_ReturnsDataExport(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().ListAllByUser(gomock.Any(), userID).Return([]practice.Practice{*p}, nil)

	email := mustEmail(t, testEmail)
	identitySvc := services.NewIdentityService(email,
		practices,
		mocks.NewMockDeletionRequestRepository(ctrl),
		mocks.NewMockAccessLogRepository(ctrl),
	)
	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		identitySvc,
		testQuizService(ctrl),
		testTutorService(ctrl),
	)

	rec := serve(userID, srv.ExportData, httptest.NewRequest(http.MethodGet, "/me/data/export", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body DataExport
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Email != "student@example.com" {
		t.Fatalf("email = %q, want student@example.com", body.Email)
	}
	if body.UserId.String() != userID.String() {
		t.Fatalf("user_id = %s, want %s", body.UserId, userID)
	}
	if body.GeneratedAt.IsZero() {
		t.Fatal("generated_at is zero, want a timestamp")
	}
	if len(body.Practices) != 1 || body.Practices[0].Id.String() != p.ID.String() {
		t.Fatalf("practices = %+v, want the user's practice", body.Practices)
	}
}

func TestServer_ExportData_RepositoryError_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().ListAllByUser(gomock.Any(), userID).Return(nil, &domain.InternalError{Field: "practice"})

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		services.NewIdentityService(mustEmail(t, testEmail), practices, mocks.NewMockDeletionRequestRepository(ctrl), mocks.NewMockAccessLogRepository(ctrl)),
		testQuizService(ctrl),
		testTutorService(ctrl),
	)

	rec := serve(userID, srv.ExportData, httptest.NewRequest(http.MethodGet, "/me/data/export", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestServer_DeleteData_NoPending_Returns202(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	deletions := mocks.NewMockDeletionRequestRepository(ctrl)
	deletions.EXPECT().HasPending(gomock.Any(), userID).Return(false, nil)
	deletions.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		services.NewIdentityService(mustEmail(t, testEmail), mocks.NewMockPracticeRepository(ctrl), deletions, mocks.NewMockAccessLogRepository(ctrl)),
		testQuizService(ctrl),
		testTutorService(ctrl),
	)

	rec := serve(userID, srv.DeleteData, httptest.NewRequest(http.MethodDelete, "/me/data", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
}

func TestServer_DeleteData_AlreadyPending_Returns202Idempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	deletions := mocks.NewMockDeletionRequestRepository(ctrl)
	deletions.EXPECT().HasPending(gomock.Any(), userID).Return(true, nil)
	// No Append expected: the call is idempotent.

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		services.NewIdentityService(mustEmail(t, testEmail), mocks.NewMockPracticeRepository(ctrl), deletions, mocks.NewMockAccessLogRepository(ctrl)),
		testQuizService(ctrl),
		testTutorService(ctrl),
	)

	rec := serve(userID, srv.DeleteData, httptest.NewRequest(http.MethodDelete, "/me/data", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
}

func TestServer_GetAccessLog_ReturnsPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	event, err := identity.NewAccessEvent(userID, "GET", "/practices", "", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("NewAccessEvent() error = %v", err)
	}

	accessLog := mocks.NewMockAccessLogRepository(ctrl)
	accessLog.EXPECT().ListByUser(gomock.Any(), userID, 20, 0).Return([]identity.AccessEvent{*event}, 1, nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		services.NewIdentityService(mustEmail(t, testEmail), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockDeletionRequestRepository(ctrl), accessLog),
		testQuizService(ctrl),
		testTutorService(ctrl),
	)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) {
		srv.GetAccessLog(w, r, GetAccessLogParams{})
	}, httptest.NewRequest(http.MethodGet, "/me/access-log", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body AccessLog
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Total != 1 || len(body.Items) != 1 {
		t.Fatalf("body = %+v, want one entry", body)
	}
	if body.Items[0].Action != "GET" {
		t.Fatalf("action = %q, want GET", body.Items[0].Action)
	}
	if body.Items[0].ResourceType == nil || *body.Items[0].ResourceType != "/practices" {
		t.Fatalf("resource_type = %v, want /practices", body.Items[0].ResourceType)
	}
}

func TestServer_MeEndpoints_MissingUser_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv := newTestServer(ctrl, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))

	cases := map[string]http.HandlerFunc{
		"access-log": func(w http.ResponseWriter, r *http.Request) { srv.GetAccessLog(w, r, GetAccessLogParams{}) },
		"delete":     srv.DeleteData,
		"export":     srv.ExportData,
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
		})
	}
}

func TestServer_UpdatePractice_Valid_Returns200(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodPatch, "/practices/"+p.ID.String(),
		strings.NewReader(`{"draft_text":"The dog has escaped."}`))

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) { srv.UpdatePractice(w, r, wireID(p.ID)) }, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body Practice
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.DraftText != "The dog has escaped." {
		t.Fatalf("draft = %q", body.DraftText)
	}
}

func TestServer_GetPractice_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.NotFoundError{Field: "practice"})

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodGet, "/practices/"+id.String(), nil)

	rec := serve(domain.MustNewID(), func(w http.ResponseWriter, r *http.Request) { srv.GetPractice(w, r, wireID(id)) }, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestServer_AnalyzePractice_AlreadyAnalyzing_Returns409(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)
	p.Status = practice.PracticeStatusAnalyzing

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	srv := newTestServer(ctrl, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))
	req := httptest.NewRequest(http.MethodPost, "/practices/"+p.ID.String()+"/analyze", nil)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) { srv.AnalyzePractice(w, r, wireID(p.ID)) }, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestGeneratedRouter_GetPractice_RoutesAndBindsParam(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(nil, &domain.NotFoundError{Field: "analysis"})

	srv := newTestServer(ctrl, practices, analyses, mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockOutbox(ctrl))

	router := chi.NewRouter()
	router.Use(httpx.UserResolver(userID))
	HandlerFromMux(srv, router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/practices/"+p.ID.String(), nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var body PracticeDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Id.String() != p.ID.String() {
		t.Fatalf("id = %s, want %s", body.Id, p.ID)
	}
}

func TestServer_CreateQuizQuestion_ReturnsQuestion(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(&analysis.Analysis{
		ID:         domain.MustNewID(),
		PracticeID: p.ID,
		Status:     analysis.AnalysisStatusCompleted,
		Fragments:  []analysis.Fragment{{SourceES: "x", UserDraft: "y", Correction: "z"}},
	}, nil)
	questioner := mocks.NewMockTutorQuestioner(ctrl)
	questioner.EXPECT().Question(gomock.Any(), gomock.Any()).Return(
		analysis.QuizQuestion{Kind: analysis.QuizKindFill, Prompt: "I bet ___ the races."}, nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		testIdentityService(ctrl),
		services.NewQuizService(practices, analyses, questioner),
		testTutorService(ctrl),
	)

	router := chi.NewRouter()
	router.Use(httpx.UserResolver(userID))
	HandlerFromMux(srv, router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/practices/"+p.ID.String()+"/quiz", strings.NewReader(`{"fragment_index":0}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var question QuizQuestion
	if err := json.Unmarshal(rec.Body.Bytes(), &question); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if question.Kind != QuizQuestionKind(analysis.QuizKindFill) || question.Prompt != "I bet ___ the races." {
		t.Fatalf("question = %+v", question)
	}
}

func TestServer_EvaluateQuizAnswer_ReturnsEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := draftPractice(userID)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(&analysis.Analysis{
		ID:         domain.MustNewID(),
		PracticeID: p.ID,
		Status:     analysis.AnalysisStatusCompleted,
		Fragments:  []analysis.Fragment{{SourceES: "x", UserDraft: "y", Correction: "z"}},
	}, nil)
	questioner := mocks.NewMockTutorQuestioner(ctrl)
	questioner.EXPECT().Evaluate(gomock.Any(), gomock.Any()).Return(
		analysis.QuizEvaluation{Correct: true, Feedback: "¡Correcto!"}, nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		testIdentityService(ctrl),
		services.NewQuizService(practices, analyses, questioner),
		testTutorService(ctrl),
	)

	router := chi.NewRouter()
	router.Use(httpx.UserResolver(userID))
	HandlerFromMux(srv, router)

	body := `{"fragment_index":0,"question":"I bet ___ the races.","answer":"on"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/practices/"+p.ID.String()+"/quiz/answer", strings.NewReader(body))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var evaluation QuizEvaluation
	if err := json.Unmarshal(rec.Body.Bytes(), &evaluation); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !evaluation.Correct || evaluation.Feedback != "¡Correcto!" {
		t.Fatalf("evaluation = %+v", evaluation)
	}
}

// TestGeneratedRouter_DocsEndpoints_NotFound encodes AP-MR9 (7.2.3): the backend
// must not serve interactive API docs. The generated mux only mounts the
// contract operations, so every docs path resolves to 404.
func TestGeneratedRouter_DocsEndpoints_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	srv := newTestServer(ctrl,
		mocks.NewMockPracticeRepository(ctrl),
		mocks.NewMockAnalysisRepository(ctrl),
		mocks.NewMockErrorMetricRepository(ctrl),
		mocks.NewMockOutbox(ctrl),
	)

	router := chi.NewRouter()
	router.Use(httpx.UserResolver(userID))
	HandlerFromMux(srv, router)

	for _, path := range []string{"/docs", "/docs/", "/openapi.json", "/swagger", "/scalar"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404 (AP-MR9: docs no expuesto)", path, rec.Code)
		}
	}
}
