package analysis_test

import (
	"encoding/json"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

func TestAnalysisStatus_IsValid_KnownValues_ReturnsTrue(t *testing.T) {
	for _, status := range []analysis.AnalysisStatus{
		analysis.AnalysisStatusPending,
		analysis.AnalysisStatusCompleted,
		analysis.AnalysisStatusFailed,
	} {
		if !status.IsValid() {
			t.Fatalf("IsValid(%q) = false, want true", status)
		}
	}
}

func TestAnalysisStatus_IsValid_Unknown_ReturnsFalse(t *testing.T) {
	if analysis.AnalysisStatus("running").IsValid() {
		t.Fatal(`IsValid("running") = true, want false`)
	}
}

func TestAnalysisStatus_String_ReturnsWireValue(t *testing.T) {
	if got := analysis.AnalysisStatusPending.String(); got != "pending" {
		t.Fatalf("String() = %q, want %q", got, "pending")
	}
}

func TestAnalysisStatus_MarshalJSON_UsesLowercaseValue(t *testing.T) {
	raw, err := json.Marshal(analysis.AnalysisStatusCompleted)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(raw) != `"completed"` {
		t.Fatalf("Marshal() = %s, want %q", raw, "completed")
	}
}
