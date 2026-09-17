package practice_test

import (
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestSourceText_NewSourceText_Valid_ReturnsValue(t *testing.T) {
	got, err := practice.NewSourceText("  El gato duerme.  ")
	if err != nil {
		t.Fatalf("NewSourceText() error = %v", err)
	}
	if got.String() != "  El gato duerme.  " {
		t.Fatalf("String() = %q, want the raw text preserved", got.String())
	}
}

func TestSourceText_NewSourceText_Blank_ReturnsValidationError(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		_, err := practice.NewSourceText(raw)
		if err == nil {
			t.Fatalf("NewSourceText(%q): want error, got nil", raw)
		}
		var target *domain.ValidationError
		if !errors.As(err, &target) {
			t.Fatalf("NewSourceText(%q) error = %T, want *domain.ValidationError", raw, err)
		}
		if target.Field != "source_text" {
			t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "source_text")
		}
	}
}
