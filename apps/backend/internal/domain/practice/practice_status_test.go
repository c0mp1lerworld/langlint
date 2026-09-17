package practice_test

import (
	"encoding/json"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestPracticeStatus_IsValid_KnownValues_ReturnsTrue(t *testing.T) {
	for _, status := range []practice.PracticeStatus{
		practice.PracticeStatusDraft,
		practice.PracticeStatusAnalyzing,
		practice.PracticeStatusCompleted,
		practice.PracticeStatusFailed,
	} {
		if !status.IsValid() {
			t.Fatalf("IsValid(%q) = false, want true", status)
		}
	}
}

func TestPracticeStatus_IsValid_Unknown_ReturnsFalse(t *testing.T) {
	if practice.PracticeStatus("archived").IsValid() {
		t.Fatal(`IsValid("archived") = true, want false`)
	}
}

func TestPracticeStatus_String_ReturnsWireValue(t *testing.T) {
	if got := practice.PracticeStatusDraft.String(); got != "draft" {
		t.Fatalf("String() = %q, want %q", got, "draft")
	}
}

func TestPracticeStatus_MarshalJSON_UsesLowercaseValue(t *testing.T) {
	raw, err := json.Marshal(practice.PracticeStatusAnalyzing)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(raw) != `"analyzing"` {
		t.Fatalf("Marshal() = %s, want %q", raw, "analyzing")
	}
}
