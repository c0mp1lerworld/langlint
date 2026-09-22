package tutor_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

var testNow = time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

func TestWeaknessEntry_NewWeaknessEntry_Valid_NormalizesTimestampToUTC(t *testing.T) {
	entry, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityModerate, 3, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry() error = %v", err)
	}
	if entry.Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Code = %q, want %q", entry.Code, domain.ErrorPatternCodeWordOrder)
	}
	if entry.Severity != domain.ErrorPatternSeverityModerate {
		t.Fatalf("Severity = %q, want %q", entry.Severity, domain.ErrorPatternSeverityModerate)
	}
	if entry.Count != 3 {
		t.Fatalf("Count = %d, want 3", entry.Count)
	}
	if !entry.LastSeenAt.Equal(testNow) {
		t.Fatalf("LastSeenAt = %v, want %v", entry.LastSeenAt, testNow)
	}
}

func TestWeaknessEntry_NewWeaknessEntry_UnknownCode_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewWeaknessEntry("spelling", domain.ErrorPatternSeverityModerate, 1, testNow)
	assertValidationError(t, err, "weakness.code")
}

func TestWeaknessEntry_NewWeaknessEntry_UnknownSeverity_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, "fatal", 1, testNow)
	assertValidationError(t, err, "weakness.severity")
}

func TestWeaknessEntry_NewWeaknessEntry_NonPositiveCount_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityModerate, 0, testNow)
	assertValidationError(t, err, "weakness.count")
}

func TestWeaknessEntry_NewWeaknessEntry_ZeroLastSeenAt_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityModerate, 1, time.Time{})
	assertValidationError(t, err, "weakness.last_seen_at")
}

func TestWeaknessProfile_NewWeaknessProfile_Valid_SortsByDescendingCount(t *testing.T) {
	low, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityMinor, 1, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry(low) error = %v", err)
	}
	high, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodePrepositionInfinitive, domain.ErrorPatternSeverityCritical, 5, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry(high) error = %v", err)
	}

	profile, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{low, high})
	if err != nil {
		t.Fatalf("NewWeaknessProfile() error = %v", err)
	}
	if len(profile.Entries) != 2 {
		t.Fatalf("Entries len = %d, want 2", len(profile.Entries))
	}
	if profile.Entries[0].Code != domain.ErrorPatternCodePrepositionInfinitive {
		t.Fatalf("Entries[0].Code = %q, want %q", profile.Entries[0].Code, domain.ErrorPatternCodePrepositionInfinitive)
	}
	if profile.Entries[1].Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Entries[1].Code = %q, want %q", profile.Entries[1].Code, domain.ErrorPatternCodeWordOrder)
	}
}

func TestWeaknessProfile_NewWeaknessProfile_Empty_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewWeaknessProfile(nil)
	assertValidationError(t, err, "weakness_profile.entries")
}

func TestWeaknessProfile_NewWeaknessProfile_InvalidEntry_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{{Code: "spelling"}})
	assertValidationError(t, err, "weakness.code")
}

func TestWeaknessProfile_Weakest_ReturnsHighestCount(t *testing.T) {
	low, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityMinor, 1, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry(low) error = %v", err)
	}
	high, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodePrepositionInfinitive, domain.ErrorPatternSeverityCritical, 5, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry(high) error = %v", err)
	}
	profile, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{low, high})
	if err != nil {
		t.Fatalf("NewWeaknessProfile() error = %v", err)
	}

	if got := profile.Weakest(); got.Code != domain.ErrorPatternCodePrepositionInfinitive {
		t.Fatalf("Weakest().Code = %q, want %q", got.Code, domain.ErrorPatternCodePrepositionInfinitive)
	}
}

func TestWeaknessProfile_MarshalJSON_UsesSnakeCase(t *testing.T) {
	entry, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityModerate, 3, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry() error = %v", err)
	}
	profile, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{entry})
	if err != nil {
		t.Fatalf("NewWeaknessProfile() error = %v", err)
	}

	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := got["entries"]; !ok {
		t.Fatalf("payload %s missing key %q", raw, "entries")
	}
	entries, ok := got["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("payload %s entries = %v, want 1 element", raw, got["entries"])
	}
	entryMap, ok := entries[0].(map[string]any)
	if !ok {
		t.Fatalf("payload %s entries[0] is not an object", raw)
	}
	for _, key := range []string{"code", "severity", "count", "last_seen_at"} {
		if _, ok := entryMap[key]; !ok {
			t.Fatalf("payload %s entry missing snake_case key %q", raw, key)
		}
	}
}

func assertValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.ValidationError", err)
	}
	if target.Field != field {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, field)
	}
}
