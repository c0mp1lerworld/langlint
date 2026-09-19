package event_handlers

import (
	"context"
	"errors"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// AnalysisCompletedHandler closes the loop of a successful analysis: it
// materializes the practice status (analyzing -> completed) and updates the
// error-pattern metrics of the analytics bounded context (manifest §5.1). It is
// idempotent: it skips practices that are no longer analyzing.
type AnalysisCompletedHandler struct {
	practices storage.PracticeRepository
	metrics   storage.ErrorMetricRepository
	now       func() time.Time
}

// NewAnalysisCompletedHandler wires the handler through its ports.
func NewAnalysisCompletedHandler(practices storage.PracticeRepository, metrics storage.ErrorMetricRepository) *AnalysisCompletedHandler {
	return &AnalysisCompletedHandler{practices: practices, metrics: metrics, now: time.Now}
}

// Handle materializes the completed analysis into the practice and analytics.
func (h *AnalysisCompletedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	completed, ok := analysisCompleted(event)
	if !ok {
		return nil
	}

	p, err := h.practices.GetByID(ctx, completed.PracticeID)
	if err != nil {
		var notFound *domain.NotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return err
	}
	if p.Status != practice.PracticeStatusAnalyzing {
		return nil
	}

	// Metrics first, status last: if the metric update fails the event can be
	// retried while the practice is still analyzing, so no data is lost.
	if err := h.upsertMetrics(ctx, completed); err != nil {
		return err
	}

	if err := p.MarkCompleted(h.now()); err != nil {
		return err
	}
	return h.practices.Save(ctx, p)
}

// upsertMetrics increments the absolute count of each error pattern in the
// event for every aggregation window.
func (h *AnalysisCompletedHandler) upsertMetrics(ctx context.Context, event domain.AnalysisCompleted) error {
	increments := countByCode(event.ErrorPatterns)
	if len(increments) == 0 {
		return nil
	}

	for _, window := range []analytics.Window{analytics.WindowDay, analytics.WindowWeek, analytics.WindowMonth} {
		current, err := h.metrics.ListByUser(ctx, event.UserID, window)
		if err != nil {
			return err
		}

		counts := make(map[domain.ErrorPatternCode]int, len(current))
		for _, metric := range current {
			counts[metric.Code] = metric.Count
		}

		for code, delta := range increments {
			metric := &analytics.ErrorMetric{
				UserID:     event.UserID,
				Code:       code,
				Window:     window,
				Count:      counts[code] + delta,
				LastSeenAt: h.now(),
			}
			if err := h.metrics.Upsert(ctx, metric); err != nil {
				return err
			}
		}
	}
	return nil
}

// countByCode counts the occurrences of each error code in the event payload.
func countByCode(patterns []domain.ErrorPattern) map[domain.ErrorPatternCode]int {
	counts := make(map[domain.ErrorPatternCode]int, len(patterns))
	for _, pattern := range patterns {
		counts[pattern.Code]++
	}
	return counts
}
