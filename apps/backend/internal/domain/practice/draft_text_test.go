package practice_test

import (
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestDraftText_NewDraftText_Valid_ReturnsValue(t *testing.T) {
	got, err := practice.NewDraftText("  The cat sleeps.  ")
	if err != nil {
		t.Fatalf("NewDraftText() error = %v", err)
	}
	if got.String() != "  The cat sleeps.  " {
		t.Fatalf("String() = %q, want the raw text preserved", got.String())
	}
}

func TestDraftText_NewDraftText_Blank_ReturnsValidationError(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		_, err := practice.NewDraftText(raw)
		if err == nil {
			t.Fatalf("NewDraftText(%q): want error, got nil", raw)
		}
		var target *domain.ValidationError
		if !errors.As(err, &target) {
			t.Fatalf("NewDraftText(%q) error = %T, want *domain.ValidationError", raw, err)
		}
		if target.Field != "draft_text" {
			t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "draft_text")
		}
	}
}
