package analytics_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestProgressMetric_NewProgressMetric_Valid_ComputesAccuracy(t *testing.T) {
	userID := domain.MustNewID()

	m, err := analytics.NewProgressMetric(userID, analytics.WindowWeek, 4, 1)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}
	if m.UserID != userID {
		t.Fatalf("UserID = %s, want %s", m.UserID, userID)
	}
	if m.Window != analytics.WindowWeek {
		t.Fatalf("Window = %q, want %q", m.Window, analytics.WindowWeek)
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
	m, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowDay, 0, 0)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}
	if m.Accuracy != 1 {
		t.Fatalf("Accuracy = %v, want 1", m.Accuracy)
	}
}

func TestProgressMetric_NewProgressMetric_ErrorsExceedFragments_ClampsToZero(t *testing.T) {
	m, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowMonth, 2, 5)
	if err != nil {
		t.Fatalf("NewProgressMetric() error = %v", err)
	}
	if m.Accuracy != 0 {
		t.Fatalf("Accuracy = %v, want 0", m.Accuracy)
	}
}

func TestProgressMetric_NewProgressMetric_UnknownWindow_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewProgressMetric(domain.MustNewID(), "year", 4, 1)
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

func TestProgressMetric_NewProgressMetric_NegativeTotalFragments_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, -1, 0)
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
	_, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, 4, -1)
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
	m, err := analytics.NewProgressMetric(domain.MustNewID(), analytics.WindowWeek, 4, 1)
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
	for _, key := range []string{"user_id", "window", "total_fragments", "error_count", "accuracy"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 5 {
		t.Fatalf("payload %s has %d keys, want exactly 5", raw, len(got))
	}
}
