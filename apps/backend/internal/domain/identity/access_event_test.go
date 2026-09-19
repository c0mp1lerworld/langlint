package identity_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

func TestAccessEvent_NewAccessEvent_SetsFields(t *testing.T) {
	userID := domain.MustNewID()
	occurredAt := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC)

	event, err := identity.NewAccessEvent(userID, "GET", "/practices", "", occurredAt)
	if err != nil {
		t.Fatalf("NewAccessEvent() error = %v", err)
	}

	if event.UserID != userID {
		t.Fatalf("UserID = %s, want %s", event.UserID, userID)
	}
	if event.Action != "GET" {
		t.Fatalf("Action = %q, want GET", event.Action)
	}
	if event.ResourceType != "/practices" {
		t.Fatalf("ResourceType = %q, want /practices", event.ResourceType)
	}
	if !event.OccurredAt.Equal(occurredAt) {
		t.Fatalf("OccurredAt = %v, want %v", event.OccurredAt, occurredAt)
	}
	if !event.ID.IsValid() {
		t.Fatalf("ID = %s, want a valid UUID v7", event.ID)
	}
}

func TestAccessEvent_NewAccessEvent_TrimsTextFields(t *testing.T) {
	event, err := identity.NewAccessEvent(domain.MustNewID(), "  POST  ", "  /practices  ", "  id-1  ", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewAccessEvent() error = %v", err)
	}
	if event.Action != "POST" || event.ResourceType != "/practices" || event.ResourceID != "id-1" {
		t.Fatalf("fields = (%q, %q, %q), want trimmed", event.Action, event.ResourceType, event.ResourceID)
	}
}

func TestAccessEvent_NewAccessEvent_ZeroUserID_ReturnsValidationError(t *testing.T) {
	_, err := identity.NewAccessEvent(domain.ID{}, "GET", "", "", time.Now().UTC())
	if err == nil {
		t.Fatal("NewAccessEvent() with zero user id: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewAccessEvent() error = %T, want *domain.ValidationError", err)
	}
}

func TestAccessEvent_NewAccessEvent_EmptyAction_ReturnsValidationError(t *testing.T) {
	for _, action := range []string{"", "   "} {
		_, err := identity.NewAccessEvent(domain.MustNewID(), action, "", "", time.Now().UTC())
		if err == nil {
			t.Fatalf("NewAccessEvent(action=%q): want error, got nil", action)
		}
		var target *domain.ValidationError
		if !errors.As(err, &target) {
			t.Fatalf("NewAccessEvent(action=%q) error = %T, want *domain.ValidationError", action, err)
		}
	}
}

func TestAccessEvent_MarshalJSON_UsesSnakeCaseAndNoTenantID(t *testing.T) {
	event, err := identity.NewAccessEvent(domain.MustNewID(), "GET", "/me/access-log", "", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewAccessEvent() error = %v", err)
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "user_id", "action", "resource_type", "resource_id", "occurred_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if _, ok := got["tenant_id"]; ok {
		t.Fatalf("payload %s must not contain tenant_id (AP1)", raw)
	}
}
