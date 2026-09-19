package services

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestAnalyticsService_ErrorPatternStats_DelegatesToRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	want := []analytics.ErrorMetric{{
		UserID: userID,
		Code:   domain.ErrorPatternCodeWordOrder,
		Window: analytics.WindowWeek,
		Count:  3,
	}}

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, analytics.WindowWeek).Return(want, nil)

	svc := NewAnalyticsService(metrics)
	got, err := svc.ErrorPatternStats(context.Background(), userID, analytics.WindowWeek)
	if err != nil {
		t.Fatalf("ErrorPatternStats() error = %v", err)
	}
	if len(got) != 1 || got[0].Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("ErrorPatternStats() = %v, want %v", got, want)
	}
}
