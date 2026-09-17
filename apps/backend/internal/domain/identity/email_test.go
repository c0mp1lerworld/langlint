package identity_test

import (
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

func TestEmail_NewEmail_Valid_ReturnsEmail(t *testing.T) {
	got, err := identity.NewEmail("student@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}
	if got.String() != "student@example.com" {
		t.Fatalf("String() = %q, want %q", got.String(), "student@example.com")
	}
}

func TestEmail_NewEmail_Invalid_ReturnsValidationError(t *testing.T) {
	_, err := identity.NewEmail("not-an-email")
	if err == nil {
		t.Fatal("NewEmail() of invalid address: want error, got nil")
	}

	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewEmail() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "email" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "email")
	}
}

func TestEmail_NewEmail_Empty_ReturnsValidationError(t *testing.T) {
	_, err := identity.NewEmail("   ")
	if err == nil {
		t.Fatal("NewEmail() of blank address: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewEmail() error = %T, want *domain.ValidationError", err)
	}
}
