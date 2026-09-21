package services

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// AnalyticsService serves the analytics read models of the API. It depends on
// ports only (A2).
type AnalyticsService struct {
	metrics  storage.ErrorMetricRepository
	progress storage.ProgressRepository
}

// NewAnalyticsService wires the read models through their ports.
func NewAnalyticsService(metrics storage.ErrorMetricRepository, progress storage.ProgressRepository) *AnalyticsService {
	return &AnalyticsService{metrics: metrics, progress: progress}
}

// ErrorPatternStats returns the user's error-pattern metrics for a window.
func (s *AnalyticsService) ErrorPatternStats(ctx context.Context, userID domain.ID, window analytics.Window) ([]analytics.ErrorMetric, error) {
	return s.metrics.ListByUser(ctx, userID, window)
}

// ProgressSeries returns the user's temporal progress for a window. The series
// is derived on read from the raw samples and bucketed by the pure domain
// function (PRODUCT_DOMAIN §7.1).
func (s *AnalyticsService) ProgressSeries(ctx context.Context, userID domain.ID, window analytics.Window) ([]analytics.ProgressMetric, error) {
	samples, err := s.progress.ListSamplesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return analytics.BuildProgressSeries(userID, window, samples)
}
