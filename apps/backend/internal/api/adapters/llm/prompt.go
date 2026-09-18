package llm

import (
	"fmt"
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// prompt is the provider-agnostic instruction pair sent to the LLM (4.1.2).
// Keeping it free of SDK types lets the prompt construction be unit-tested
// without network access.
type prompt struct {
	System string
	User   string
}

// errorPatternDescriptions mirrors the taxonomy of PRODUCT_DOMAIN §4.6. The
// codes come from the domain constants so the prompt can never drift from the
// contract; the descriptions disambiguate them for the model.
var errorPatternDescriptions = []struct {
	code        domain.ErrorPatternCode
	description string
}{
	{domain.ErrorPatternCodeInfinitiveConjugation, "confusion between an infinitive and a conjugated verb"},
	{domain.ErrorPatternCodePassiveVoiceMisuse, "wrong use of the passive voice"},
	{domain.ErrorPatternCodeIdiomLiteralTranslation, "literal translation of an idiom"},
	{domain.ErrorPatternCodePrepositionInfinitive, "malformed preposition + infinitive"},
	{domain.ErrorPatternCodePronounPossession, "confusion between pronoun and possessive"},
	{domain.ErrorPatternCodeFalseFriend, "lexical false friend"},
	{domain.ErrorPatternCodeLexicalChoice, "wrong vocabulary choice or collocation that is not a false friend"},
	{domain.ErrorPatternCodeWordOrder, "wrong syntactic order"},
	{domain.ErrorPatternCodeTenseAgreement, "verb tense agreement"},
}

// errorPatternSeverities mirrors the severities of PRODUCT_DOMAIN §4.6.
var errorPatternSeverities = []domain.ErrorPatternSeverity{
	domain.ErrorPatternSeverityMinor,
	domain.ErrorPatternSeverityModerate,
	domain.ErrorPatternSeverityCritical,
}

const systemPrompt = `You are a native English teacher and professional editor helping a Spanish-speaking learner improve their written English.
Analyze the learner's English draft against the Spanish source, sentence by sentence. Cover the entire text: produce one fragment per sentence or meaningful clause, in the original order.
For every fragment return the exact original Spanish sentence, the exact original English draft, a direct English correction, a review of the target verb, a lexical clarification and a deep grammar explanation.

The correction is in English. Write target_verb_review, lexical_clarification, grammar_explanation and the error notes in Spanish (the learner's native language).
Classify every mistake with the error pattern taxonomy below.
Use ONLY the codes below; never invent new ones. Use lexical_choice for vocabulary or collocation choices (including non-idiomatic collocations), and word_order only for genuinely reordered words.

Error pattern codes (code: meaning):%s

Error pattern severities (severity): %s

Respond with ONLY a JSON array. Do not wrap it in markdown code fences and do not add any prose.
Each element must be an object with exactly these keys:
- source_es (string): the base Spanish sentence, copied verbatim from the source text.
- user_draft (string): the learner's English draft for that sentence, copied verbatim from the draft.
- correction (string): the corrected English sentence.
- target_verb_review (string): review of the target verb(s) used, in Spanish.
- lexical_clarification (string): lexical notes, in Spanish.
- grammar_explanation (string): the grammar rule behind the mistake, in Spanish.
- error_patterns (array): the mistakes found. Use an empty array when there is no mistake. Each item is an object with keys code (one of the codes above), severity (one of the severities above) and note (a short explanation in Spanish).`

// PromptText returns the exact system and user messages that Extract sends to
// the provider. It is exported so operators and the manual `cmd/llmcheck`
// runner can audit the prompt without calling the API.
func PromptText(req ports.ExtractRequest) (system, user string) {
	p := buildPrompt(req)
	return p.System, p.User
}

// buildPrompt constructs the anonymized prompt from the extraction request
// (4.1.2). The request fields are already anonymized upstream (A8); this
// function does not touch personal data.
func buildPrompt(req ports.ExtractRequest) prompt {
	return prompt{
		System: fmt.Sprintf(systemPrompt, errorPatternCatalog(), joinValues(errorPatternSeverities)),
		User:   buildUserPrompt(req),
	}
}

// errorPatternCatalog renders the code taxonomy as a bullet list with meanings.
func errorPatternCatalog() string {
	var b strings.Builder
	for _, item := range errorPatternDescriptions {
		fmt.Fprintf(&b, "\n- %s: %s", item.code, item.description)
	}
	return b.String()
}

// buildUserPrompt renders the practice data as the user message.
func buildUserPrompt(req ports.ExtractRequest) string {
	var b strings.Builder

	b.WriteString("Source text (Spanish):\n")
	b.WriteString(req.SourceText)
	b.WriteString("\n\nLearner draft (English):\n")
	b.WriteString(req.DraftText)
	b.WriteString("\n\nTarget rules to practice:")

	if len(req.TargetRules) == 0 {
		b.WriteString("\n- (none)")
		return b.String()
	}

	for _, rule := range req.TargetRules {
		fmt.Fprintf(&b, "\n- verb: %s", rule.Verb)
		if rule.Tense != "" {
			fmt.Fprintf(&b, " | tense: %s", rule.Tense)
		}
		if rule.Note != "" {
			fmt.Fprintf(&b, " | note: %s", rule.Note)
		}
	}

	return b.String()
}

// joinValues renders a slice of string-based values as a comma-separated list.
func joinValues[T ~string](values []T) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = string(value)
	}
	return strings.Join(parts, ", ")
}
