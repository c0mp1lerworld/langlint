package tutor_test

import (
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func TestTrap_NewTrap_Valid_TrimsDescription(t *testing.T) {
	trap, err := tutor.NewTrap(domain.ErrorPatternCodePrepositionInfinitive, "  Using \"to\" before a gerund  ")
	if err != nil {
		t.Fatalf("NewTrap() error = %v", err)
	}
	if trap.Code != domain.ErrorPatternCodePrepositionInfinitive {
		t.Fatalf("Code = %q, want %q", trap.Code, domain.ErrorPatternCodePrepositionInfinitive)
	}
	if trap.Description != "Using \"to\" before a gerund" {
		t.Fatalf("Description = %q, want trimmed", trap.Description)
	}
}

func TestTrap_NewTrap_UnknownCode_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewTrap("spelling", "a description")
	assertValidationError(t, err, "trap.code")
}

func TestTrap_NewTrap_EmptyDescription_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewTrap(domain.ErrorPatternCodePrepositionInfinitive, "   ")
	assertValidationError(t, err, "trap.description")
}
