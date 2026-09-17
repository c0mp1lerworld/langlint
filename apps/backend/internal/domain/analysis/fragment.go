package analysis

import "github.com/c0mp1lerworld/langlint/backend/internal/domain"

// Fragment is the atomic unit of an analysis (PRODUCT_DOMAIN §5.2): the base
// sentence, the user's draft, the correction and the deep explanation.
type Fragment struct {
	SourceES             string                `json:"source_es"`
	UserDraft            string                `json:"user_draft"`
	Correction           string                `json:"correction"`
	TargetVerbReview     string                `json:"target_verb_review"`
	LexicalClarification string                `json:"lexical_clarification"`
	GrammarExplanation   string                `json:"grammar_explanation"`
	ErrorPatterns        []domain.ErrorPattern `json:"error_patterns"`
}
