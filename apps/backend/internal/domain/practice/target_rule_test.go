package practice_test

import (
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestTargetRule_NewTargetRule_Valid_TrimsFields(t *testing.T) {
	got, err := practice.NewTargetRule("  sleep ", " present ", " irregular verb ")
	if err != nil {
		t.Fatalf("NewTargetRule() error = %v", err)
	}
	if got.Verb != "sleep" {
		t.Fatalf("Verb = %q, want %q", got.Verb, "sleep")
	}
	if got.Tense != "present" {
		t.Fatalf("Tense = %q, want %q", got.Tense, "present")
	}
	if got.Note != "irregular verb" {
		t.Fatalf("Note = %q, want %q", got.Note, "irregular verb")
	}
}

func TestTargetRule_NewTargetRule_BlankVerb_ReturnsValidationError(t *testing.T) {
	for _, verb := range []string{"", "   "} {
		_, err := practice.NewTargetRule(verb, "present", "")
		if err == nil {
			t.Fatalf("NewTargetRule(%q): want error, got nil", verb)
		}
		var target *domain.ValidationError
		if !errors.As(err, &target) {
			t.Fatalf("NewTargetRule(%q) error = %T, want *domain.ValidationError", verb, err)
		}
		if target.Field != "target_rules.verb" {
			t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "target_rules.verb")
		}
	}
}
