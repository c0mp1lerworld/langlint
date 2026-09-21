package practice_test

import (
	"errors"
	"strings"
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

func TestSourceText_NewSourceText_MaxLength_IsAccepted(t *testing.T) {
	if _, err := practice.NewSourceText(strings.Repeat("a", 2000)); err != nil {
		t.Fatalf("NewSourceText(2000 runes) error = %v, want nil", err)
	}
}

func TestSourceText_NewSourceText_TooLong_ReturnsValidationError(t *testing.T) {
	_, err := practice.NewSourceText(strings.Repeat("a", 2001))
	if err == nil {
		t.Fatal("NewSourceText(2001 runes): want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "source_text" {
		t.Fatalf("ValidationError.Field = %q, want source_text", target.Field)
	}
}
