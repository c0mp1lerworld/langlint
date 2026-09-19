package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
)

func pattern(code domain.ErrorPatternCode) domain.ErrorPattern {
	return domain.ErrorPattern{Code: code, Severity: domain.ErrorPatternSeverityMinor}
}

func completed(userID domain.ID, at time.Time, codes ...domain.ErrorPatternCode) storage.CompletedAnalysis {
	patterns := make([]domain.ErrorPattern, 0, len(codes))
	for _, code := range codes {
		patterns = append(patterns, pattern(code))
	}
	return storage.CompletedAnalysis{
		UserID:      userID,
		Fragments:   []analysis.Fragment{{ErrorPatterns: patterns}},
		CompletedAt: at,
	}
}

// metricKey identifies a metric by window and code for order-independent
// assertions.
type metricKey struct {
	window analytics.Window
	code   domain.ErrorPatternCode
}

// collect indexes the metrics by window and code.
func collect(t *testing.T, metrics []*analytics.ErrorMetric) map[metricKey]*analytics.ErrorMetric {
	t.Helper()
	byKey := make(map[metricKey]*analytics.ErrorMetric, len(metrics))
	for _, m := range metrics {
		byKey[metricKey{window: m.Window, code: m.Code}] = m
	}
	return byKey
}

func TestRefreshAggregatesService_Refresh_CountsPatternsAcrossAllWindows(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	base := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return([]storage.CompletedAnalysis{
		completed(userID, base, domain.ErrorPatternCodeWordOrder, domain.ErrorPatternCodeWordOrder),
		completed(userID, base.Add(time.Hour), domain.ErrorPatternCodeTenseAgreement),
	}, nil)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ReplaceAll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got []*analytics.ErrorMetric) error {
			if len(got) != 6 {
				t.Fatalf("ReplaceAll() got %d metrics, want 6 (2 codes x 3 windows)", len(got))
			}
			byKey := collect(t, got)
			want := map[domain.ErrorPatternCode]struct {
				count    int
				lastSeen time.Time
			}{
				domain.ErrorPatternCodeWordOrder:      {count: 2, lastSeen: base},
				domain.ErrorPatternCodeTenseAgreement: {count: 1, lastSeen: base.Add(time.Hour)},
			}
			for _, w := range analytics.AllWindows {
				for code, exp := range want {
					m, ok := byKey[metricKey{window: w, code: code}]
					if !ok {
						t.Fatalf("missing metric for window %s code %s", w, code)
					}
					if m.Count != exp.count {
						t.Fatalf("window %s code %s count = %d, want %d", w, code, m.Count, exp.count)
					}
					if !m.LastSeenAt.Equal(exp.lastSeen) {
						t.Fatalf("window %s code %s last_seen_at = %v, want %v", w, code, m.LastSeenAt, exp.lastSeen)
					}
				}
			}
			return nil
		},
	)

	svc := NewRefreshAggregatesService(source, metrics)
	got, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if got != 6 {
		t.Fatalf("Refresh() = %d, want 6", got)
	}
}

func TestRefreshAggregatesService_Refresh_NoAnalyses_ReplacesWithEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return(nil, nil)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ReplaceAll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got []*analytics.ErrorMetric) error {
			if len(got) != 0 {
				t.Fatalf("ReplaceAll() got %d metrics, want 0", len(got))
			}
			return nil
		},
	)

	svc := NewRefreshAggregatesService(source, metrics)
	got, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if got != 0 {
		t.Fatalf("Refresh() = %d, want 0", got)
	}
}

func TestRefreshAggregatesService_Refresh_MultipleUsers_KeepsMetricsSeparate(t *testing.T) {
	ctrl := gomock.NewController(t)
	first, second := domain.MustNewID(), domain.MustNewID()
	now := time.Now().UTC()

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return([]storage.CompletedAnalysis{
		completed(first, now, domain.ErrorPatternCodeWordOrder, domain.ErrorPatternCodeWordOrder),
		completed(second, now, domain.ErrorPatternCodeWordOrder),
	}, nil)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ReplaceAll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got []*analytics.ErrorMetric) error {
			byUser := make(map[domain.ID]int)
			for _, m := range got {
				byUser[m.UserID] = m.Count
			}
			if byUser[first] != 2 || byUser[second] != 1 {
				t.Fatalf("counts by user = %v, want first=2 second=1", byUser)
			}
			return nil
		},
	)

	svc := NewRefreshAggregatesService(source, metrics)
	if _, err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
}

func TestRefreshAggregatesService_Refresh_SourceError_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return(nil, errors.New("source down"))

	metrics := mocks.NewMockErrorMetricRepository(ctrl)

	svc := NewRefreshAggregatesService(source, metrics)
	if _, err := svc.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() with failing source: want error, got nil")
	}
}

func TestRefreshAggregatesService_Refresh_InvalidStoredCode_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return([]storage.CompletedAnalysis{
		completed(domain.MustNewID(), time.Now().UTC(), domain.ErrorPatternCode("bogus")),
	}, nil)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)

	svc := NewRefreshAggregatesService(source, metrics)
	if _, err := svc.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() with an unknown stored code: want error, got nil")
	}
}

func TestRefreshAggregatesService_Refresh_ReplaceError_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return([]storage.CompletedAnalysis{
		completed(domain.MustNewID(), time.Now().UTC(), domain.ErrorPatternCodeWordOrder),
	}, nil)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ReplaceAll(gomock.Any(), gomock.Any()).Return(errors.New("write down"))

	svc := NewRefreshAggregatesService(source, metrics)
	if _, err := svc.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() with failing ReplaceAll: want error, got nil")
	}
}
