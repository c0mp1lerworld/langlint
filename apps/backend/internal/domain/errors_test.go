package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestValidationError_Error_IncludesFieldAndMessage(t *testing.T) {
	err := &domain.ValidationError{Field: "email", Message: "must be a valid email address"}

	msg := err.Error()
	if msg == "" {
		t.Fatal("Error() must not be empty")
	}
	for _, want := range []string{"email", "must be a valid email address"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Error() = %q, want it to contain %q", msg, want)
		}
	}
}

func TestValidationError_AsDomainError(t *testing.T) {
	var target *domain.ValidationError
	err := error(&domain.ValidationError{Field: "email", Message: "invalid"})

	if !errors.As(err, &target) {
		t.Fatal("errors.As must match *domain.ValidationError")
	}
}

func TestValidationError_Error_WithoutField_ReturnsMessage(t *testing.T) {
	err := &domain.ValidationError{Message: "invalid input"}

	if got := err.Error(); got != "invalid input" {
		t.Fatalf("Error() = %q, want %q", got, "invalid input")
	}
}

func TestInvalidStateError_Error_IncludesFieldAndMessage(t *testing.T) {
	err := &domain.InvalidStateError{Field: "status", Message: "must be draft to be edited"}

	msg := err.Error()
	for _, want := range []string{"status", "must be draft to be edited"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Error() = %q, want it to contain %q", msg, want)
		}
	}
}

func TestInvalidStateError_Error_WithoutField_ReturnsMessage(t *testing.T) {
	err := &domain.InvalidStateError{Message: "invalid state"}

	if got := err.Error(); got != "invalid state" {
		t.Fatalf("Error() = %q, want %q", got, "invalid state")
	}
}

func TestInvalidStateError_AsDomainError(t *testing.T) {
	var target *domain.InvalidStateError
	err := error(&domain.InvalidStateError{Field: "status", Message: "invalid"})

	if !errors.As(err, &target) {
		t.Fatal("errors.As must match *domain.InvalidStateError")
	}
}
