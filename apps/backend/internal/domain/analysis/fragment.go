package analysis

import "github.com/c0mp1lerworld/langlint/backend/internal/domain"

// Fragment is the atomic unit of an analysis (PRODUCT_DOMAIN §5.2): the base
// sentence, the user's draft, the correction and the deep, structured
// explanation ("Beyond Correction", §1.2).
type Fragment struct {
	SourceES             string                `json:"source_es"`
	UserDraft            string                `json:"user_draft"`
	Correction           string                `json:"correction"`
	TargetVerbReview     TargetVerbReview      `json:"target_verb_review"`
	LexicalClarification LexicalClarification  `json:"lexical_clarification"`
	GrammarExplanation   GrammarExplanation    `json:"grammar_explanation"`
	ErrorPatterns        []domain.ErrorPattern `json:"error_patterns"`
}

// TargetVerbReview is the structured review of the target verb: it names the
// rule and explains the why instead of emitting a generic paragraph.
type TargetVerbReview struct {
	Verb         string   `json:"verb"`
	CorrectForm  string   `json:"correct_form"`
	Rule         string   `json:"rule"`
	Why          string   `json:"why"`
	ESContrast   string   `json:"es_contrast"`
	Alternatives []string `json:"alternatives"`
}

// LexicalClarification is the structured lexical note for a vocabulary or
// collocation choice.
type LexicalClarification struct {
	Term         string   `json:"term"`
	Meaning      string   `json:"meaning"`
	WhyWrong     string   `json:"why_wrong"`
	Alternatives []string `json:"alternatives"`
}

// GrammarExplanation is the deep grammar rule behind a mistake, broken into the
// ingredients a learner needs: name, why, construction, counterexample,
// exceptions and the Spanish contrast.
type GrammarExplanation struct {
	RuleName       string `json:"rule_name"`
	Explanation    string `json:"explanation"`
	Construction   string `json:"construction"`
	Counterexample string `json:"counterexample"`
	Exception      string `json:"exception"`
	ESContrast     string `json:"es_contrast"`
}
