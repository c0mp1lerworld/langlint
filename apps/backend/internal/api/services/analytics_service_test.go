package services

import (
	"context"
	"testing"
	"time"

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

	svc := NewAnalyticsService(metrics, nil)
	got, err := svc.ErrorPatternStats(context.Background(), userID, analytics.WindowWeek)
	if err != nil {
		t.Fatalf("ErrorPatternStats() error = %v", err)
	}
	if len(got) != 1 || got[0].Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("ErrorPatternStats() = %v, want %v", got, want)
	}
}

func TestAnalyticsService_ProgressSeries_BucketsSamples(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	samples := []analytics.ProgressSample{
		{CompletedAt: time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC), TotalFragments: 3, ErrorCount: 1},
		{CompletedAt: time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC), TotalFragments: 2, ErrorCount: 0},
	}

	progress := mocks.NewMockProgressRepository(ctrl)
	progress.EXPECT().ListSamplesByUser(gomock.Any(), userID).Return(samples, nil)

	svc := NewAnalyticsService(nil, progress)
	got, err := svc.ProgressSeries(context.Background(), userID, analytics.WindowWeek)
	if err != nil {
		t.Fatalf("ProgressSeries() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(ProgressSeries) = %d, want 1", len(got))
	}
	if got[0].TotalFragments != 5 || got[0].ErrorCount != 1 {
		t.Fatalf("ProgressSeries()[0] = %+v, want total=5 errors=1", got[0])
	}
}

func TestAnalyticsService_ProgressSeries_RepositoryError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	progress := mocks.NewMockProgressRepository(ctrl)
	progress.EXPECT().ListSamplesByUser(gomock.Any(), userID).Return(nil, &domain.InternalError{Field: "progress", Message: "database error"})

	svc := NewAnalyticsService(nil, progress)
	if _, err := svc.ProgressSeries(context.Background(), userID, analytics.WindowDay); err == nil {
		t.Fatal("ProgressSeries() with failing repository: want error, got nil")
	}
}
