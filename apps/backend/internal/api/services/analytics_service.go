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
	metrics storage.ErrorMetricRepository
}

// NewAnalyticsService wires the read model through its port.
func NewAnalyticsService(metrics storage.ErrorMetricRepository) *AnalyticsService {
	return &AnalyticsService{metrics: metrics}
}

// ErrorPatternStats returns the user's error-pattern metrics for a window.
func (s *AnalyticsService) ErrorPatternStats(ctx context.Context, userID domain.ID, window analytics.Window) ([]analytics.ErrorMetric, error) {
	return s.metrics.ListByUser(ctx, userID, window)
}
