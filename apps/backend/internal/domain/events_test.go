package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestIdentityIssued_EventName_ReturnsConstant(t *testing.T) {
	event := domain.IdentityIssued{UserID: domain.MustNewID(), Version: 1}

	if got := event.EventName(); got != domain.EventNameIdentityIssued {
		t.Fatalf("EventName() = %q, want %q", got, domain.EventNameIdentityIssued)
	}
}

func TestIdentityIssued_ImplementsDomainEvent(t *testing.T) {
	var event domain.DomainEvent = domain.IdentityIssued{UserID: domain.MustNewID(), Version: 1}

	if event.EventName() == "" {
		t.Fatal("IdentityIssued must satisfy DomainEvent with a non-empty name")
	}
}

func TestPracticeCreated_EventName_ReturnsConstant(t *testing.T) {
	event := domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}

	if got := event.EventName(); got != domain.EventNamePracticeCreated {
		t.Fatalf("EventName() = %q, want %q", got, domain.EventNamePracticeCreated)
	}
}

func TestPracticeCreated_ImplementsDomainEvent(t *testing.T) {
	var event domain.DomainEvent = domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}

	if event.EventName() == "" {
		t.Fatal("PracticeCreated must satisfy DomainEvent with a non-empty name")
	}
}

func TestPracticeCreated_MarshalJSON_UsesSnakeCase(t *testing.T) {
	event := domain.PracticeCreated{PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"practice_id", "user_id", "version"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if _, ok := got["PracticeID"]; ok {
		t.Fatalf("payload %s leaked Go field name %q", raw, "PracticeID")
	}
}

func TestAnalysisCompleted_EventName_ReturnsConstant(t *testing.T) {
	event := domain.AnalysisCompleted{AnalysisID: domain.MustNewID(), PracticeID: domain.MustNewID(), UserID: domain.MustNewID(), Version: 1}

	if got := event.EventName(); got != domain.EventNameAnalysisCompleted {
		t.Fatalf("EventName() = %q, want %q", got, domain.EventNameAnalysisCompleted)
	}
}

func TestAnalysisCompleted_ImplementsDomainEvent(t *testing.T) {
	var event domain.DomainEvent = domain.AnalysisCompleted{AnalysisID: domain.MustNewID(), Version: 1}

	if event.EventName() == "" {
		t.Fatal("AnalysisCompleted must satisfy DomainEvent with a non-empty name")
	}
}

func TestAnalysisCompleted_MarshalJSON_UsesSnakeCase(t *testing.T) {
	event := domain.AnalysisCompleted{
		AnalysisID:    domain.MustNewID(),
		PracticeID:    domain.MustNewID(),
		UserID:        domain.MustNewID(),
		ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityMinor}},
		Version:       1,
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"analysis_id", "practice_id", "user_id", "error_patterns", "version"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if _, ok := got["AnalysisID"]; ok {
		t.Fatalf("payload %s leaked Go field name %q", raw, "AnalysisID")
	}
}

func TestAnalysisFailed_EventName_ReturnsConstant(t *testing.T) {
	event := domain.AnalysisFailed{AnalysisID: domain.MustNewID(), PracticeID: domain.MustNewID(), Reason: "timeout", Version: 1}

	if got := event.EventName(); got != domain.EventNameAnalysisFailed {
		t.Fatalf("EventName() = %q, want %q", got, domain.EventNameAnalysisFailed)
	}
}

func TestAnalysisFailed_MarshalJSON_UsesSnakeCase(t *testing.T) {
	event := domain.AnalysisFailed{AnalysisID: domain.MustNewID(), PracticeID: domain.MustNewID(), Reason: "timeout", Version: 1}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"analysis_id", "practice_id", "reason", "version"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
}

func TestIdentityIssued_MarshalJSON_UsesSnakeCase(t *testing.T) {
	event := domain.IdentityIssued{UserID: domain.MustNewID(), Version: 1}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := got["user_id"]; !ok {
		t.Fatalf("payload %s missing snake_case key %q", raw, "user_id")
	}
	if _, ok := got["version"]; !ok {
		t.Fatalf("payload %s missing key %q", raw, "version")
	}
	if _, ok := got["UserID"]; ok {
		t.Fatalf("payload %s leaked Go field name %q", raw, "UserID")
	}
}

func TestNewEvent_KnownType_ReturnsConcreteEvent(t *testing.T) {
	for _, name := range []string{
		domain.EventNameIdentityIssued,
		domain.EventNamePracticeCreated,
		domain.EventNameAnalysisCompleted,
		domain.EventNameAnalysisFailed,
	} {
		t.Run(name, func(t *testing.T) {
			event, ok := domain.NewEvent(name)
			if !ok {
				t.Fatalf("NewEvent(%q) ok = false, want true", name)
			}
			if got := event.EventName(); got != name {
				t.Fatalf("NewEvent(%q).EventName() = %q, want %q", name, got, name)
			}
		})
	}
}

func TestNewEvent_UnknownType_ReturnsFalse(t *testing.T) {
	event, ok := domain.NewEvent("unknown.event")

	if ok {
		t.Fatal("NewEvent(unknown) ok = true, want false")
	}
	if event != nil {
		t.Fatalf("NewEvent(unknown) = %v, want nil", event)
	}
}

func TestOutboxEvent_MarshalJSON_UsesSnakeCase(t *testing.T) {
	published := time.Unix(0, 0).UTC()
	event := domain.OutboxEvent{
		ID:          domain.MustNewID(),
		EventType:   domain.EventNamePracticeCreated,
		Payload:     []byte(`{}`),
		CreatedAt:   published,
		PublishedAt: &published,
		Attempts:    2,
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "event_type", "payload", "created_at", "published_at", "attempts"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if _, ok := got["EventType"]; ok {
		t.Fatalf("payload %s leaked Go field name %q", raw, "EventType")
	}
}
