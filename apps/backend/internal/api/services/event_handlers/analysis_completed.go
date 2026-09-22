package event_handlers

import (
	"context"
	"errors"
	"time"

	portsevents "github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// AnalysisCompletedHandler closes the loop of a successful analysis: it
// materializes the practice status (analyzing -> completed), updates the
// error-pattern metrics of the analytics bounded context (manifest §5.1) and,
// when a frequency threshold is crossed, emits WeaknessDetected to feed the
// tutor bounded context (checklist 8.2). It is idempotent: it skips practices
// that are no longer analyzing.
type AnalysisCompletedHandler struct {
	practices storage.PracticeRepository
	metrics   storage.ErrorMetricRepository
	outbox    portsevents.Outbox
	now       func() time.Time
}

// NewAnalysisCompletedHandler wires the handler through its ports.
func NewAnalysisCompletedHandler(practices storage.PracticeRepository, metrics storage.ErrorMetricRepository, outbox portsevents.Outbox) *AnalysisCompletedHandler {
	return &AnalysisCompletedHandler{practices: practices, metrics: metrics, outbox: outbox, now: time.Now}
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
// event for every aggregation window, and emits WeaknessDetected for the codes
// whose frequency reached the threshold (8.2.1, 8.2.3).
func (h *AnalysisCompletedHandler) upsertMetrics(ctx context.Context, event domain.AnalysisCompleted) error {
	increments := countByCode(event.ErrorPatterns)
	if len(increments) == 0 {
		return nil
	}
	severities := severityByCode(event.ErrorPatterns)

	for _, window := range analytics.AllWindows {
		current, err := h.metrics.ListByUser(ctx, event.UserID, window)
		if err != nil {
			return err
		}

		counts := make(map[domain.ErrorPatternCode]int, len(current))
		for _, metric := range current {
			counts[metric.Code] = metric.Count
		}

		merged := make(map[domain.ErrorPatternCode]int, len(counts)+len(increments))
		for code, count := range counts {
			merged[code] = count
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
			merged[code] = counts[code] + delta
		}

		if err := h.emitWeakness(ctx, event.UserID, window, merged, severities); err != nil {
			return err
		}
	}
	return nil
}

// emitWeakness appends a WeaknessDetected event for the codes whose aggregated
// frequency reached the threshold, carrying the severity seen in the analysis.
func (h *AnalysisCompletedHandler) emitWeakness(ctx context.Context, userID domain.ID, window domain.Window, merged map[domain.ErrorPatternCode]int, severities map[domain.ErrorPatternCode]domain.ErrorPatternSeverity) error {
	detected := analytics.DetectWeakness(merged, analytics.WeaknessThreshold)
	if len(detected) == 0 {
		return nil
	}

	patterns := make([]domain.ErrorPattern, 0, len(detected))
	for _, code := range detected {
		patterns = append(patterns, domain.ErrorPattern{Code: code, Severity: severities[code]})
	}

	return h.outbox.Append(ctx, domain.WeaknessDetected{
		UserID:        userID,
		ErrorPatterns: patterns,
		Window:        window,
		Version:       1,
	})
}

// countByCode counts the occurrences of each error code in the event payload.
func countByCode(patterns []domain.ErrorPattern) map[domain.ErrorPatternCode]int {
	counts := make(map[domain.ErrorPatternCode]int, len(patterns))
	for _, pattern := range patterns {
		counts[pattern.Code]++
	}
	return counts
}

// severityByCode maps each error code to the highest severity observed in the
// patterns (minor < moderate < critical).
func severityByCode(patterns []domain.ErrorPattern) map[domain.ErrorPatternCode]domain.ErrorPatternSeverity {
	severities := make(map[domain.ErrorPatternCode]domain.ErrorPatternSeverity, len(patterns))
	for _, pattern := range patterns {
		if current, ok := severities[pattern.Code]; !ok || severityRank(pattern.Severity) > severityRank(current) {
			severities[pattern.Code] = pattern.Severity
		}
	}
	return severities
}

func severityRank(s domain.ErrorPatternSeverity) int {
	switch s {
	case domain.ErrorPatternSeverityCritical:
		return 3
	case domain.ErrorPatternSeverityModerate:
		return 2
	case domain.ErrorPatternSeverityMinor:
		return 1
	default:
		return 0
	}
}
