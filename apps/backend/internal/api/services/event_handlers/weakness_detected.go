package event_handlers

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

// WeaknessDetectedHandler is the tutor bounded context's subscriber to the
// WeaknessDetected event (checklist 8.2.2). It assembles the aggregated weakness
// into a tutor.WeaknessProfile from the analytics metrics; generating the study
// session from that profile is the next step (8.3). It is idempotent: it reads
// only aggregates and produces no side effect.
type WeaknessDetectedHandler struct {
	metrics storage.ErrorMetricRepository
}

// NewWeaknessDetectedHandler wires the handler through its ports.
func NewWeaknessDetectedHandler(metrics storage.ErrorMetricRepository) *WeaknessDetectedHandler {
	return &WeaknessDetectedHandler{metrics: metrics}
}

// Handle builds the weakness profile for the detected patterns. The profile is
// the aggregated input the tutor consumes (PRODUCT_DOMAIN §12.1); session
// generation and persistence arrive in 8.3.
func (h *WeaknessDetectedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	weakness, ok := weaknessDetected(event)
	if !ok {
		return nil
	}

	metrics, err := h.metrics.ListByUser(ctx, weakness.UserID, weakness.Window)
	if err != nil {
		return err
	}

	_, err = buildWeaknessProfile(metrics, weakness.ErrorPatterns)
	return err
}

// buildWeaknessProfile enriches the detected patterns with their aggregated
// frequency and recency from the metrics, and returns a validated profile. A
// pattern with no matching metric is skipped (the metric may not be visible yet
// under at-least-once delivery).
func buildWeaknessProfile(metrics []analytics.ErrorMetric, patterns []domain.ErrorPattern) (tutor.WeaknessProfile, error) {
	byCode := make(map[domain.ErrorPatternCode]analytics.ErrorMetric, len(metrics))
	for _, metric := range metrics {
		byCode[metric.Code] = metric
	}

	entries := make([]tutor.WeaknessEntry, 0, len(patterns))
	for _, pattern := range patterns {
		metric, ok := byCode[pattern.Code]
		if !ok {
			continue
		}
		entry, err := tutor.NewWeaknessEntry(pattern.Code, pattern.Severity, metric.Count, metric.LastSeenAt)
		if err != nil {
			return tutor.WeaknessProfile{}, err
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return tutor.WeaknessProfile{}, nil
	}
	return tutor.NewWeaknessProfile(entries)
}
