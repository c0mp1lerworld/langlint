package llm

import (
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// validateFragments enforces the PRODUCT_DOMAIN §5.2 schema on the LLM output
// before it can be persisted (4.2.2). Strict Structured Outputs already makes
// the provider conform, but the adapter never trusts the model: every required
// text field must carry content and every error pattern must use a code and a
// severity from the taxonomy. A violation is reported as
// *domain.LLMUnavailableError so no malformed fragment reaches the domain (A5).
func validateFragments(fragments []analysis.Fragment) error {
	for _, fragment := range fragments {
		if err := validateFragment(fragment); err != nil {
			return err
		}
	}
	return nil
}

// validateFragment checks a single fragment against the §5.2 contract.
func validateFragment(fragment analysis.Fragment) error {
	required := []string{
		fragment.SourceES,
		fragment.UserDraft,
		fragment.Correction,
		fragment.TargetVerbReview,
		fragment.LexicalClarification,
		fragment.GrammarExplanation,
	}
	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			return invalidOutput()
		}
	}

	for _, pattern := range fragment.ErrorPatterns {
		if !pattern.Code.IsValid() || !pattern.Severity.IsValid() {
			return invalidOutput()
		}
	}

	return nil
}

// invalidOutput wraps a schema violation as a provider failure so the raw
// validation detail never leaks upstream (A5, A8).
func invalidOutput() error {
	return &domain.LLMUnavailableError{Message: "llm unavailable"}
}
