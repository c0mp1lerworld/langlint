package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func TestServer_CreateStudySession_Returns201(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).
		Return([]analytics.ErrorMetric{{Code: domain.ErrorPatternCodeWordOrder, Count: 6, LastSeenAt: now}}, nil)

	generator := mocks.NewMockStudySessionGenerator(ctrl)
	generator.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(ports.StudySessionContent{
		Theory: "El orden de palabras en inglés es SVO.",
		Traps:  []tutor.Trap{{Code: domain.ErrorPatternCodeWordOrder, Description: "Colocar el verbo al final."}},
		Exercises: []tutor.Exercise{
			{Kind: tutor.ExerciseKindFill, Prompt: "I ___ (never) have seen.", Answer: "have never"},
		},
	}, nil)

	sessions := mocks.NewMockStudySessionRepository(ctrl)
	sessions.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		testIdentityService(ctrl),
		testQuizService(ctrl),
		services.NewStudySessionService(metrics, sessions, generator),
	)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) {
		srv.CreateStudySession(w, r, CreateStudySessionParams{})
	}, httptest.NewRequest(http.MethodPost, "/study-sessions", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}

	var body StudySession
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.UserId.String() != userID.String() {
		t.Fatalf("user_id = %s, want %s", body.UserId, userID)
	}
	if body.Status != StudySessionStatusGenerated || body.Theory == "" {
		t.Fatalf("body = %+v", body)
	}
	if len(body.Weaknesses) != 1 || len(body.Traps) != 1 || len(body.Exercises) != 1 {
		t.Fatalf("body lists = weaknesses=%d traps=%d exercises=%d, want 1 each",
			len(body.Weaknesses), len(body.Traps), len(body.Exercises))
	}
}

func TestServer_CreateStudySession_NoWeakness_Returns409(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).Return(nil, nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		testIdentityService(ctrl),
		testQuizService(ctrl),
		services.NewStudySessionService(metrics, mocks.NewMockStudySessionRepository(ctrl), mocks.NewMockStudySessionGenerator(ctrl)),
	)

	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) {
		srv.CreateStudySession(w, r, CreateStudySessionParams{})
	}, httptest.NewRequest(http.MethodPost, "/study-sessions", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestServer_CreateStudySession_InvalidWindow_Returns422(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		testIdentityService(ctrl),
		testQuizService(ctrl),
		services.NewStudySessionService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockStudySessionRepository(ctrl), mocks.NewMockStudySessionGenerator(ctrl)),
	)

	invalid := Window("year")
	rec := serve(userID, func(w http.ResponseWriter, r *http.Request) {
		srv.CreateStudySession(w, r, CreateStudySessionParams{Window: &invalid})
	}, httptest.NewRequest(http.MethodPost, "/study-sessions", nil))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body %s)", rec.Code, rec.Body.String())
	}
}
