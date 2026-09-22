package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestServer_GetQuizStats_ReturnsStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	attempts := mocks.NewMockQuizAttemptRepository(ctrl)
	attempts.EXPECT().ListByUser(gomock.Any(), userID).Return(
		[]analytics.QuizAttempt{{Correct: true}, {Correct: true}, {Correct: false}}, nil)

	srv := NewServer(
		services.NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl)),
		services.NewAnalyticsService(mocks.NewMockErrorMetricRepository(ctrl), mocks.NewMockProgressRepository(ctrl)),
		testIdentityService(ctrl),
		services.NewQuizService(mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockTutorQuestioner(ctrl), attempts),
		testTutorService(ctrl),
	)

	rec := serve(userID, srv.GetQuizStats, httptest.NewRequest(http.MethodGet, "/analytics/quiz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body QuizStats
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.TotalAttempts != 3 || body.CorrectAttempts != 2 {
		t.Fatalf("body = %+v, want total=3 correct=2", body)
	}
}
