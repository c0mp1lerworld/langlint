package main

import (
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestParseDate_ValidDay_MapsToNoonUTC(t *testing.T) {
	got, err := parseDate("2026-09-11")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}
	want := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseDate() = %v, want %v", got, want)
	}
}

func TestParseDate_Invalid_ReturnsError(t *testing.T) {
	for _, raw := range []string{"", "2026-13-40", "11/09/2026", "ayer"} {
		if _, err := parseDate(raw); err == nil {
			t.Fatalf("parseDate(%q) error = nil, want error", raw)
		}
	}
}

func TestParseEntries_EmptyArray_ReturnsError(t *testing.T) {
	if _, err := parseEntries([]byte(`[]`)); err == nil {
		t.Fatal("parseEntries([]) error = nil, want error")
	}
}

func TestParseEntries_InvalidJSON_ReturnsError(t *testing.T) {
	if _, err := parseEntries([]byte(`not json`)); err == nil {
		t.Fatal("parseEntries() error = nil, want error")
	}
}

func TestBuildPractices_Valid_SortsByDate(t *testing.T) {
	entries := []importEntry{
		{Date: "2026-09-15", SourceText: "a", DraftText: "b", TargetRules: []importRule{{Verb: "run"}}},
		{Date: "2026-09-11", SourceText: "c", DraftText: "d", TargetRules: []importRule{{Verb: "bet"}}},
	}

	practices, err := buildPractices(entries)
	if err != nil {
		t.Fatalf("buildPractices() error = %v", err)
	}
	if len(practices) != 2 {
		t.Fatalf("len = %d, want 2", len(practices))
	}
	if !practices[0].CreatedAt.Before(practices[1].CreatedAt) {
		t.Fatalf("not sorted by date: %v then %v", practices[0].CreatedAt, practices[1].CreatedAt)
	}
	if practices[0].CreatedAt.Day() != 11 {
		t.Fatalf("first = day %d, want 11", practices[0].CreatedAt.Day())
	}
}

func TestBuildPractices_StableForEqualDates(t *testing.T) {
	entries := []importEntry{
		{Date: "2026-09-11", SourceText: "primero", DraftText: "x", TargetRules: []importRule{{Verb: "a"}}},
		{Date: "2026-09-11", SourceText: "segundo", DraftText: "y", TargetRules: []importRule{{Verb: "b"}}},
	}

	practices, err := buildPractices(entries)
	if err != nil {
		t.Fatalf("buildPractices() error = %v", err)
	}
	if practices[0].Source != practice.SourceText("primero") {
		t.Fatalf("first source = %q, want primero (stable order)", practices[0].Source)
	}
}

func TestBuildPractices_EmptyRules_ReturnsError(t *testing.T) {
	entries := []importEntry{{Date: "2026-09-11", SourceText: "a", DraftText: "b", TargetRules: nil}}
	if _, err := buildPractices(entries); err == nil {
		t.Fatal("buildPractices() error = nil, want error")
	}
}

func TestBuildPractices_OverFiveRules_ReturnsError(t *testing.T) {
	rules := make([]importRule, 6)
	for i := range rules {
		rules[i] = importRule{Verb: "v"}
	}
	entries := []importEntry{{Date: "2026-09-11", SourceText: "a", DraftText: "b", TargetRules: rules}}
	if _, err := buildPractices(entries); err == nil {
		t.Fatal("buildPractices() error = nil, want error")
	}
}

func TestBuildPractices_EmptyVerb_ReturnsError(t *testing.T) {
	entries := []importEntry{{Date: "2026-09-11", SourceText: "a", DraftText: "b", TargetRules: []importRule{{Verb: "  "}}}}
	if _, err := buildPractices(entries); err == nil {
		t.Fatal("buildPractices() error = nil, want error")
	}
}

func TestBuildPractices_EmptySource_ReturnsError(t *testing.T) {
	entries := []importEntry{{Date: "2026-09-11", SourceText: "  ", DraftText: "b", TargetRules: []importRule{{Verb: "run"}}}}
	if _, err := buildPractices(entries); err == nil {
		t.Fatal("buildPractices() error = nil, want error")
	}
}
