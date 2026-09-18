package events

import (
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestMarshalEvent_ReturnsWireNameAndPayload(t *testing.T) {
	event := domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}

	name, payload, err := MarshalEvent(event)
	if err != nil {
		t.Fatalf("MarshalEvent() error = %v", err)
	}
	if name != domain.EventNamePracticeCreated {
		t.Fatalf("name = %q, want %q", name, domain.EventNamePracticeCreated)
	}
	if len(payload) == 0 {
		t.Fatal("payload must not be empty")
	}
}

func TestUnmarshalEvent_RoundTrip(t *testing.T) {
	original := domain.AnalysisCompleted{
		AnalysisID:    domain.MustNewID(),
		PracticeID:    domain.MustNewID(),
		UserID:        domain.MustNewID(),
		ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityMinor}},
		Version:       2,
	}

	name, payload, err := MarshalEvent(original)
	if err != nil {
		t.Fatalf("MarshalEvent() error = %v", err)
	}

	event, err := UnmarshalEvent(name, payload)
	if err != nil {
		t.Fatalf("UnmarshalEvent() error = %v", err)
	}
	got, ok := event.(*domain.AnalysisCompleted)
	if !ok {
		t.Fatalf("UnmarshalEvent() = %T, want *domain.AnalysisCompleted", event)
	}
	if got.AnalysisID != original.AnalysisID || got.PracticeID != original.PracticeID || got.UserID != original.UserID {
		t.Fatal("round-trip lost an identifier")
	}
	if got.Version != original.Version {
		t.Fatalf("Version = %d, want %d", got.Version, original.Version)
	}
	if len(got.ErrorPatterns) != 1 || got.ErrorPatterns[0] != original.ErrorPatterns[0] {
		t.Fatalf("ErrorPatterns = %v, want %v", got.ErrorPatterns, original.ErrorPatterns)
	}
}

func TestUnmarshalEvent_UnknownType_ReturnsError(t *testing.T) {
	if _, err := UnmarshalEvent("unknown.event", []byte(`{}`)); err == nil {
		t.Fatal("UnmarshalEvent(unknown) error = nil, want error")
	}
}

func TestUnmarshalEvent_InvalidPayload_ReturnsError(t *testing.T) {
	if _, err := UnmarshalEvent(domain.EventNamePracticeCreated, []byte(`{`)); err == nil {
		t.Fatal("UnmarshalEvent(invalid) error = nil, want error")
	}
}
