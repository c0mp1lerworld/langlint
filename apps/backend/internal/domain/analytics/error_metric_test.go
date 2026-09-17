package analytics_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

var testNow = time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

func TestErrorMetric_NewErrorMetric_Valid_StartsAtCountOne(t *testing.T) {
	userID := domain.MustNewID()

	m, err := analytics.NewErrorMetric(userID, domain.ErrorPatternCodeWordOrder, analytics.WindowWeek, testNow)
	if err != nil {
		t.Fatalf("NewErrorMetric() error = %v", err)
	}
	if m.UserID != userID {
		t.Fatalf("UserID = %s, want %s", m.UserID, userID)
	}
	if m.Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Code = %q, want %q", m.Code, domain.ErrorPatternCodeWordOrder)
	}
	if m.Window != analytics.WindowWeek {
		t.Fatalf("Window = %q, want %q", m.Window, analytics.WindowWeek)
	}
	if m.Count != 1 {
		t.Fatalf("Count = %d, want 1", m.Count)
	}
	if !m.LastSeenAt.Equal(testNow) {
		t.Fatalf("LastSeenAt = %v, want %v", m.LastSeenAt, testNow)
	}
}

func TestErrorMetric_NewErrorMetric_UnknownCode_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewErrorMetric(domain.MustNewID(), "spelling", analytics.WindowWeek, testNow)
	if err == nil {
		t.Fatal("NewErrorMetric() with unknown code: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewErrorMetric() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "code" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "code")
	}
}

func TestErrorMetric_NewErrorMetric_UnknownWindow_ReturnsValidationError(t *testing.T) {
	_, err := analytics.NewErrorMetric(domain.MustNewID(), domain.ErrorPatternCodeWordOrder, "year", testNow)
	if err == nil {
		t.Fatal("NewErrorMetric() with unknown window: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewErrorMetric() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "window" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "window")
	}
}

func TestErrorMetric_Record_IncrementsCountAndUpdatesLastSeenAt(t *testing.T) {
	m, err := analytics.NewErrorMetric(domain.MustNewID(), domain.ErrorPatternCodeWordOrder, analytics.WindowWeek, testNow)
	if err != nil {
		t.Fatalf("NewErrorMetric() error = %v", err)
	}
	later := testNow.Add(time.Hour)

	m.Record(later)
	if m.Count != 2 {
		t.Fatalf("Count = %d, want 2", m.Count)
	}
	if !m.LastSeenAt.Equal(later) {
		t.Fatalf("LastSeenAt = %v, want %v", m.LastSeenAt, later)
	}
}

func TestErrorMetric_MarshalJSON_UsesSnakeCase(t *testing.T) {
	m, err := analytics.NewErrorMetric(domain.MustNewID(), domain.ErrorPatternCodeWordOrder, analytics.WindowWeek, testNow)
	if err != nil {
		t.Fatalf("NewErrorMetric() error = %v", err)
	}

	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"user_id", "code", "window", "count", "last_seen_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 5 {
		t.Fatalf("payload %s has %d keys, want exactly 5", raw, len(got))
	}
}
