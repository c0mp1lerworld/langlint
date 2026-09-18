package llm

import (
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

func validFragment() analysis.Fragment {
	return analysis.Fragment{
		SourceES:             "El perro corre.",
		UserDraft:            "The dog run.",
		Correction:           "The dog runs.",
		TargetVerbReview:     "run (run/ran/run)",
		LexicalClarification: "correr = to run",
		GrammarExplanation:   "Third person singular takes -s.",
	}
}

func assertInvalidOutput(t *testing.T, err error) {
	t.Helper()
	var target *domain.LLMUnavailableError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.LLMUnavailableError", err)
	}
}

func TestValidateFragments_ValidFragment_ReturnsNil(t *testing.T) {
	fragment := validFragment()
	fragment.ErrorPatterns = []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeInfinitiveConjugation, Severity: domain.ErrorPatternSeverityModerate, Note: "missing -s"},
	}

	if err := validateFragments([]analysis.Fragment{fragment}); err != nil {
		t.Fatalf("validateFragments() error = %v, want nil", err)
	}
}

func TestValidateFragments_EmptySlice_ReturnsNil(t *testing.T) {
	if err := validateFragments(nil); err != nil {
		t.Fatalf("validateFragments() error = %v, want nil", err)
	}
}

func TestValidateFragments_EmptyRequiredField_ReturnsError(t *testing.T) {
	fragment := validFragment()
	fragment.GrammarExplanation = "   "

	assertInvalidOutput(t, validateFragments([]analysis.Fragment{fragment}))
}

func TestValidateFragments_UnknownErrorCode_ReturnsError(t *testing.T) {
	fragment := validFragment()
	fragment.ErrorPatterns = []domain.ErrorPattern{
		{Code: domain.ErrorPatternCode("invented_code"), Severity: domain.ErrorPatternSeverityMinor},
	}

	assertInvalidOutput(t, validateFragments([]analysis.Fragment{fragment}))
}

func TestValidateFragments_UnknownSeverity_ReturnsError(t *testing.T) {
	fragment := validFragment()
	fragment.ErrorPatterns = []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverity("severe")},
	}

	assertInvalidOutput(t, validateFragments([]analysis.Fragment{fragment}))
}
