package analytics_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestProgressMetric_NewProgressMetric_Valid_ComputesAccuracy(t *testing.T) {
	userID := domain.MustNewID()
	period := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)

	m, err := analytics.NewProgressMetric(userID, analytics.WindowWeek, period, 4, 1)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}
	if m.UserID != userID {
		t.Fatalf("UserID = %s, want %s", m.UserID, userID)
	}
	if m.Window != analytics.WindowWeek {
		t.Fatalf("Window = %q, want %q", m.Window, analytics.WindowWeek)
	}
	if !m.PeriodStart.Equal(period) {
		t.Fatalf("PeriodStart = %s, want %s", m.PeriodStart, period)
	}
	if m.TotalFragments != 4 {
		t.Fatalf("TotalFragments = %d, want 4", m.TotalFragments)
	}
	if m.ErrorCount != 1 {
		t.Fatalf("ErrorCount = %d, want 1", m.ErrorCount)
	}
	if m.Accuracy != 0.75 {
		t.Fatalf("Accuracy = %v, want 0.75", m.Accuracy)
	}
}

func TestProgressMetric_NewProgressMetric_NoFragments_AccuracyIsOne(t *testing.T) {
	m, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowDay, time.Now(), 0, 0)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}
	if m.Accuracy != 1 {
		t.Fatalf("Accuracy = %v, want 1", m.Accuracy)
	}
}

func TestProgressMetric_NewProgressMetric_ErrorsExceedFragments_ClampsToZero(t *testing.T) {
	m, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowMonth, time.Now(), 2, 5)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}
	if m.Accuracy != 0 {
		t.Fatalf("Accuracy = %v, want 0", m.Accuracy)
	}
}

func TestProgressMetric_NewProgressMetric_UnknownWindow_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewProgressMetric(domain.MustNewID(), "year", time.Now(), 4, 1)
	if err == nil {
		t.Fatal("NewProgressMetric() with unknown window: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewProgressMetric() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "window" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "window")
	}
}

func TestProgressMetric_NewProgressMetric_ZeroPeriod_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, time.Time{}, 4, 1)
	if err == nil {
		t.Fatal("NewProgressMetric() with zero period: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewProgressMetric() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "period_start" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "period_start")
	}
}

func TestProgressMetric_NewProgressMetric_NegativeTotalFragments_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, time.Now(), -1, 0)
	if err == nil {
		t.Fatal("NewProgressMetric() with negative total: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewProgressMetric() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "total_fragments" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "total_fragments")
	}
}

func TestProgressMetric_NewProgressMetric_NegativeErrorCount_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, time.Now(), 4, -1)
	if err == nil {
		t.Fatal("NewProgressMetric() with negative errors: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewProgressMetric() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "error_count" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "error_count")
	}
}

func TestProgressMetric_MarshalJSON_UsesSnakeCase(t *testing.T) {
	m, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, time.Now(), 4, 1)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}

	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"user_id", "window", "period_start", "total_fragments", "error_count", "accuracy"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 6 {
		t.Fatalf("payload %s has %d keys, want exactly 6", raw, len(got))
	}
}

func TestBucketPeriod_Day_TruncatesToMidnightUTC(t *testing.T) {
	in := time.Date(2026, 9, 17, 15, 4, 5, 123, time.FixedZone("CEST", 2*3600))

	got := analytics.BucketPeriod(in, analytics.WindowDay)
	want := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("BucketPeriod(day) = %s, want %s", got, want)
	}
}

func TestBucketPeriod_Week_TruncatesToMonday(t *testing.T) {
	// Thursday 2026-09-17 -> Monday 2026-09-14.
	in := time.Date(2026, 9, 17, 9, 30, 0, 0, time.UTC)

	got := analytics.BucketPeriod(in, analytics.WindowWeek)
	want := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("BucketPeriod(week) = %s, want %s", got, want)
	}
}

func TestBucketPeriod_Week_SundayBelongsToPreviousMonday(t *testing.T) {
	// Sunday 2026-09-20 -> Monday 2026-09-14.
	in := time.Date(2026, 9, 20, 23, 59, 0, 0, time.UTC)

	got := analytics.BucketPeriod(in, analytics.WindowWeek)
	want := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("BucketPeriod(week) = %s, want %s", got, want)
	}
}

func TestBucketPeriod_Month_TruncatesToFirstDay(t *testing.T) {
	in := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	got := analytics.BucketPeriod(in, analytics.WindowMonth)
	want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("BucketPeriod(month) = %s, want %s", got, want)
	}
}

func TestBuildProgressSeries_AggregatesAndOrdersByPeriod(t *testing.T) {
	userID := domain.MustNewID()
	samples := []analytics.ProgressSample{
		{CompletedAt: time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC), TotalFragments: 3, ErrorCount: 1},
		{CompletedAt: time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC), TotalFragments: 2, ErrorCount: 0},
		{CompletedAt: time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC), TotalFragments: 4, ErrorCount: 2},
	}

	got, err := analytics.BuildProgressSeries(userID, analytics.WindowWeek, samples)
	if err != nil {
		t.Fatalf("BuildProgressSeries() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(series) = %d, want 1 (all samples in the same week)", len(got))
	}
	point := got[0]
	if !point.PeriodStart.Equal(time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("PeriodStart = %s, want 2026-09-14", point.PeriodStart)
	}
	if point.TotalFragments != 9 {
		t.Fatalf("TotalFragments = %d, want 9", point.TotalFragments)
	}
	if point.ErrorCount != 3 {
		t.Fatalf("ErrorCount = %d, want 3", point.ErrorCount)
	}
	wantAccuracy := 1 - float64(point.ErrorCount)/float64(point.TotalFragments)
	if point.Accuracy != wantAccuracy {
		t.Fatalf("Accuracy = %v, want %v", point.Accuracy, wantAccuracy)
	}
}

func TestBuildProgressSeries_MultiplePeriods_OldestFirst(t *testing.T) {
	userID := domain.MustNewID()
	samples := []analytics.ProgressSample{
		{CompletedAt: time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC), TotalFragments: 3, ErrorCount: 1},
		{CompletedAt: time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC), TotalFragments: 2, ErrorCount: 0},
	}

	got, err := analytics.BuildProgressSeries(userID, analytics.WindowWeek, samples)
	if err != nil {
		t.Fatalf("BuildProgressSeries() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(series) = %d, want 2", len(got))
	}
	if !got[0].PeriodStart.Before(got[1].PeriodStart) {
		t.Fatalf("series not ordered oldest first: %s, %s", got[0].PeriodStart, got[1].PeriodStart)
	}
}

func TestBuildProgressSeries_NoSamples_ReturnsEmpty(t *testing.T) {
	got, err := analytics.BuildProgressSeries(domain.MustNewID(), analytics.WindowDay, nil)
	if err != nil {
		t.Fatalf("BuildProgressSeries() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(series) = %d, want 0", len(got))
	}
}

func TestBuildProgressSeries_UnknownWindow_ReturnsValidationError(t *testing.T) {
	_, err := analytics.BuildProgressSeries(domain.MustNewID(), "year", nil)
	if err == nil {
		t.Fatal("BuildProgressSeries() with unknown window: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("BuildProgressSeries() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "window" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "window")
	}
}
