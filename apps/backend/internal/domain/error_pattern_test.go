package domain_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestErrorPatternCode_IsValid_KnownValues_ReturnsTrue(t *testing.T) {
	for _, code := range []domain.ErrorPatternCode{
		domain.ErrorPatternCodeInfinitiveConjugation,
		domain.ErrorPatternCodePassiveVoiceMisuse,
		domain.ErrorPatternCodeIdiomLiteralTranslation,
		domain.ErrorPatternCodePrepositionInfinitive,
		domain.ErrorPatternCodePronounPossession,
		domain.ErrorPatternCodeFalseFriend,
		domain.ErrorPatternCodeLexicalChoice,
		domain.ErrorPatternCodeWordOrder,
		domain.ErrorPatternCodeTenseAgreement,
	} {
		if !code.IsValid() {
			t.Fatalf("IsValid(%q) = false, want true", code)
		}
	}
}

func TestErrorPatternCode_IsValid_Unknown_ReturnsFalse(t *testing.T) {
	if domain.ErrorPatternCode("spelling").IsValid() {
		t.Fatal(`IsValid("spelling") = true, want false`)
	}
}

func TestErrorPatternSeverity_IsValid_KnownValues_ReturnsTrue(t *testing.T) {
	for _, severity := range []domain.ErrorPatternSeverity{
		domain.ErrorPatternSeverityMinor,
		domain.ErrorPatternSeverityModerate,
		domain.ErrorPatternSeverityCritical,
	} {
		if !severity.IsValid() {
			t.Fatalf("IsValid(%q) = false, want true", severity)
		}
	}
}

func TestErrorPatternSeverity_IsValid_Unknown_ReturnsFalse(t *testing.T) {
	if domain.ErrorPatternSeverity("fatal").IsValid() {
		t.Fatal(`IsValid("fatal") = true, want false`)
	}
}

func TestErrorPattern_NewErrorPattern_Valid_ReturnsValue(t *testing.T) {
	got, err := domain.NewErrorPattern(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityModerate, "  sujeto al final  ")
	if err != nil {
		t.Fatalf("NewErrorPattern() error = %v", err)
	}
	if got.Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Code = %q, want %q", got.Code, domain.ErrorPatternCodeWordOrder)
	}
	if got.Severity != domain.ErrorPatternSeverityModerate {
		t.Fatalf("Severity = %q, want %q", got.Severity, domain.ErrorPatternSeverityModerate)
	}
	if got.Note != "sujeto al final" {
		t.Fatalf("Note = %q, want %q", got.Note, "sujeto al final")
	}
}

func TestErrorPattern_NewErrorPattern_UnknownCode_ReturnsValidationError(t *testing.T) {
	_, err := domain.NewErrorPattern("spelling", domain.ErrorPatternSeverityMinor, "")
	if err == nil {
		t.Fatal("NewErrorPattern() with unknown code: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewErrorPattern() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "error_pattern.code" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "error_pattern.code")
	}
}

func TestErrorPattern_NewErrorPattern_UnknownSeverity_ReturnsValidationError(t *testing.T) {
	_, err := domain.NewErrorPattern(domain.ErrorPatternCodeWordOrder, "fatal", "")
	if err == nil {
		t.Fatal("NewErrorPattern() with unknown severity: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewErrorPattern() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "error_pattern.severity" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "error_pattern.severity")
	}
}

func TestErrorPattern_MarshalJSON_UsesSnakeCase(t *testing.T) {
	pattern := domain.ErrorPattern{
		Code:     domain.ErrorPatternCodeFalseFriend,
		Severity: domain.ErrorPatternSeverityCritical,
		Note:     "actually, eventually",
	}

	raw, err := json.Marshal(pattern)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"code", "severity", "note"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
}
