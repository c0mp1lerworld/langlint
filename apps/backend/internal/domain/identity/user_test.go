package identity_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

func newTestEmail(t *testing.T, raw string) identity.Email {
	t.Helper()
	email, err := identity.NewEmail(raw)
	if err != nil {
		t.Fatalf("NewEmail(%q) error = %v", raw, err)
	}
	return email
}

func TestUser_NewUser_SetsEmailAndCreatedAt(t *testing.T) {
	createdAt := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)
	email := newTestEmail(t, "student@example.com")

	user, err := identity.NewUser(email, createdAt)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if user.Email != email {
		t.Fatalf("Email = %q, want %q", user.Email, email)
	}
	if !user.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", user.CreatedAt, createdAt)
	}
	if !user.ID.IsValid() {
		t.Fatalf("ID = %s, want a valid UUID v7", user.ID)
	}
}

func TestUser_NewUser_GeneratesUniqueIDs(t *testing.T) {
	createdAt := time.Now().UTC()
	email := newTestEmail(t, "student@example.com")

	a, err := identity.NewUser(email, createdAt)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	b, err := identity.NewUser(email, createdAt)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if a.ID == b.ID {
		t.Fatalf("two users share ID %s", a.ID)
	}
}

func TestUser_MarshalJSON_HasNoTenantID(t *testing.T) {
	user, err := identity.NewUser(newTestEmail(t, "student@example.com"), time.Now().UTC())
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, forbidden := range []string{"tenant_id", "TenantID"} {
		if _, ok := got[forbidden]; ok {
			t.Fatalf("portable identity payload %s must not contain %q (AP1)", raw, forbidden)
		}
	}
}

func TestUser_MarshalJSON_UsesSnakeCase(t *testing.T) {
	user, err := identity.NewUser(newTestEmail(t, "student@example.com"), time.Now().UTC())
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "email", "created_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 3 {
		t.Fatalf("payload %s has %d keys, want exactly 3", raw, len(got))
	}
}

func TestUser_NewUser_ZeroEmail_ReturnsValidationError(t *testing.T) {
	_, err := identity.NewUser(identity.Email(""), time.Now().UTC())
	if err == nil {
		t.Fatal("NewUser() with zero email: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewUser() error = %T, want *domain.ValidationError", err)
	}
}
