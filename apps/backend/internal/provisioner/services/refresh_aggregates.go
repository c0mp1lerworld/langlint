// Package services holds the batch use cases of the provisioner entry point.
// Like the api services, they depend on ports only (A2) and never open SQL
// transactions themselves (AP8): the repositories own the transaction detail.
package services

import (
	"context"
	"sort"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
)

// RefreshAggregatesService rebuilds the materialized analytics from the
// append-only source of truth (A4). Analytics are derived data: when they drift,
// they are reconciled, never patched.
type RefreshAggregatesService struct {
	source  storage.AnalyticsSourceRepository
	metrics storage.ErrorMetricRepository
}

// NewRefreshAggregatesService wires the reconciliation through its ports.
func NewRefreshAggregatesService(source storage.AnalyticsSourceRepository, metrics storage.ErrorMetricRepository) *RefreshAggregatesService {
	return &RefreshAggregatesService{source: source, metrics: metrics}
}

// Refresh replaces every error metric by counting the error patterns of the
// completed analyses. It returns how many metrics were materialized.
func (s *RefreshAggregatesService) Refresh(ctx context.Context) (int, error) {
	analyses, err := s.source.ListCompletedAnalyses(ctx)
	if err != nil {
		return 0, err
	}

	metrics, err := rebuildErrorMetrics(analyses)
	if err != nil {
		return 0, err
	}

	if err := s.metrics.ReplaceAll(ctx, metrics); err != nil {
		return 0, err
	}
	return len(metrics), nil
}

// errorMetricKey identifies an error metric by its natural key.
type errorMetricKey struct {
	user domain.ID
	code domain.ErrorPatternCode
}

// rebuildErrorMetrics counts the error patterns of the completed analyses and
// derives one metric per (user, code, window), using the most recent analysis as
// last_seen_at. The result is sorted for deterministic reconciliation.
func rebuildErrorMetrics(analyses []storage.CompletedAnalysis) ([]*analytics.ErrorMetric, error) {
	counts := make(map[errorMetricKey]int)
	lastSeen := make(map[errorMetricKey]time.Time)

	for _, a := range analyses {
		for _, fragment := range a.Fragments {
			for _, pattern := range fragment.ErrorPatterns {
				key := errorMetricKey{user: a.UserID, code: pattern.Code}
				counts[key]++
				if a.CompletedAt.After(lastSeen[key]) {
					lastSeen[key] = a.CompletedAt
				}
			}
		}
	}

	metrics := make([]*analytics.ErrorMetric, 0, len(counts)*len(analytics.AllWindows))
	for key, count := range counts {
		for _, window := range analytics.AllWindows {
			metric, err := analytics.NewErrorMetric(key.user, key.code, window, lastSeen[key])
			if err != nil {
				return nil, err
			}
			metric.Count = count
			metrics = append(metrics, metric)
		}
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].UserID != metrics[j].UserID {
			return metrics[i].UserID.String() < metrics[j].UserID.String()
		}
		if metrics[i].Window != metrics[j].Window {
			return metrics[i].Window.String() < metrics[j].Window.String()
		}
		return metrics[i].Code < metrics[j].Code
	})

	return metrics, nil
}
