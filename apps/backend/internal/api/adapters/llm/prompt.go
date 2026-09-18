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

// errorPatternCodes mirrors the taxonomy of PRODUCT_DOMAIN §4.6. It is derived
// from the domain constants so the prompt can never drift from the contract.
var errorPatternCodes = []domain.ErrorPatternCode{
	domain.ErrorPatternCodeInfinitiveConjugation,
	domain.ErrorPatternCodePassiveVoiceMisuse,
	domain.ErrorPatternCodeIdiomLiteralTranslation,
	domain.ErrorPatternCodePrepositionInfinitive,
	domain.ErrorPatternCodePronounPossession,
	domain.ErrorPatternCodeFalseFriend,
	domain.ErrorPatternCodeWordOrder,
	domain.ErrorPatternCodeTenseAgreement,
}

// errorPatternSeverities mirrors the severities of PRODUCT_DOMAIN §4.6.
var errorPatternSeverities = []domain.ErrorPatternSeverity{
	domain.ErrorPatternSeverityMinor,
	domain.ErrorPatternSeverityModerate,
	domain.ErrorPatternSeverityCritical,
}

const systemPrompt = `You are a native English teacher and professional editor helping a Spanish-speaking learner.
Analyze the learner's English draft against the Spanish source, sentence by sentence.
For every fragment return: the base Spanish sentence, the learner's draft, a direct English correction, a review of the target verb, a lexical clarification and a deep grammar explanation that teaches the underlying rule.
Classify every mistake with the error pattern taxonomy below.

Error pattern codes (code): %s
Error pattern severities (severity): %s

Respond with ONLY a JSON array. Do not wrap it in markdown code fences and do not add any prose.
Each element must be an object with exactly these keys:
- source_es (string): the base Spanish sentence.
- user_draft (string): the learner's English draft for that sentence.
- correction (string): the corrected English sentence.
- target_verb_review (string): review of the target verb(s).
- lexical_clarification (string): lexical notes.
- grammar_explanation (string): the grammar rule behind the mistake.
- error_patterns (array): zero or more objects with keys code (one of the codes above), severity (one of the severities above) and note (string).`

// buildPrompt constructs the anonymized prompt from the extraction request
// (4.1.2). The request fields are already anonymized upstream (A8); this
// function does not touch personal data.
func buildPrompt(req ports.ExtractRequest) prompt {
	return prompt{
		System: fmt.Sprintf(systemPrompt, joinValues(errorPatternCodes), joinValues(errorPatternSeverities)),
		User:   buildUserPrompt(req),
	}
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
