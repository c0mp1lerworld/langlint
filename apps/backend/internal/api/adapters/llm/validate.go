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
	if !nonEmpty(fragment.SourceES, fragment.UserDraft, fragment.Correction) {
		return invalidOutput()
	}
	if overCorrects(fragment) {
		return invalidOutput()
	}
	for _, review := range fragment.TargetVerbReviews {
		if !validTargetVerbReview(review) {
			return invalidOutput()
		}
	}
	for _, clarification := range fragment.LexicalClarifications {
		if !validLexicalClarification(clarification) {
			return invalidOutput()
		}
	}
	for _, explanation := range fragment.GrammarExplanations {
		if !validGrammarExplanation(explanation) {
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

// validTargetVerbReview requires the rule, the why and the Spanish contrast to
// carry content. Alternatives may be empty but must not contain blanks.
func validTargetVerbReview(review analysis.TargetVerbReview) bool {
	return nonEmpty(review.Verb, review.CorrectForm, review.Rule, review.Why, review.ESContrast) &&
		validAlternatives(review.Alternatives)
}

// validLexicalClarification requires the term, its meaning and the reason the
// learner's choice does not fit.
func validLexicalClarification(clarification analysis.LexicalClarification) bool {
	return nonEmpty(clarification.Term, clarification.Meaning, clarification.WhyWrong) &&
		validAlternatives(clarification.Alternatives)
}

// validGrammarExplanation requires every pedagogical ingredient of the rule.
func validGrammarExplanation(explanation analysis.GrammarExplanation) bool {
	return nonEmpty(
		explanation.RuleName,
		explanation.Explanation,
		explanation.Construction,
		explanation.Counterexample,
		explanation.Exception,
		explanation.ESContrast,
	)
}

// nonEmpty reports whether every value carries non-whitespace content.
func nonEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

// overCorrects reports whether the correction spans more sentences than the
// learner's draft segment, which would mean the model corrected content that
// belongs to other segments (over-correction guard). A run-on draft may
// legitimately be corrected into two sentences, hence the +1 tolerance.
func overCorrects(fragment analysis.Fragment) bool {
	return len(SplitSentences(fragment.Correction)) > len(SplitSentences(fragment.UserDraft))+1
}

// validAlternatives reports whether every alternative carries content. An empty
// list is allowed (not every error has an alternative).
func validAlternatives(alternatives []string) bool {
	for _, alternative := range alternatives {
		if strings.TrimSpace(alternative) == "" {
			return false
		}
	}
	return true
}

// invalidOutput wraps a schema violation as a provider failure so the raw
// validation detail never leaks upstream (A5, A8).
func invalidOutput() error {
	return &domain.LLMUnavailableError{Message: "llm unavailable"}
}
