package domain_test

import (
	"encoding/json"
	"testing"

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
