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

// systemPrompt is the per-sentence instruction. The adapter calls the model once
// per sentence of the learner's draft so the fragments cannot drift out of
// alignment when the source and the draft have different sentence counts.
const systemPrompt = `You are a native English teacher and professional editor helping a Spanish-speaking learner improve their written English.
You receive the full Spanish source text and ONE English sentence the learner wrote as part of their draft. Produce exactly one fragment for that sentence: the Spanish source it translates, the corrected English, and a structured explanation of the mistakes that need review.
The correction is in English. Write every explanation field in Spanish (the learner's native language).
Explain as an experienced teacher would: never give a generic note. Name the rule, explain the why, show how it is built, give a counterexample, say when it does NOT apply, contrast with Spanish, and offer alternatives.
A fragment may have several entries of each kind; never merge unrelated issues into one entry. Keep every field to one short sentence (max 20 words): this is a report, not an essay.
Classify every mistake with the error pattern taxonomy below.
Use ONLY the codes below; never invent new ones. Use lexical_choice for vocabulary or collocation choices (including non-idiomatic collocations), and word_order only for genuinely reordered words.

Error pattern codes (code: meaning):%s

Error pattern severities (severity): %s

Respond with ONLY a JSON object of the shape {"fragments": [ <fragment> ]} containing exactly one fragment. Do not wrap it in markdown code fences and do not add any prose.
The fragment is an object with exactly these keys:
- source_es (string): the Spanish source that this English sentence translates, copied verbatim from the source text. Include every Spanish sentence or clause it covers.
- user_draft (string): the learner's English sentence, copied verbatim.
- correction (string): the corrected English sentence.
- target_verb_reviews (array of objects, in Spanish): one entry per target verb or verb construction in the sentence that needs review; never collapse several verb problems into one entry. Return at most three entries; if there are more, keep the three most important. Use an empty array when there is no relevant target verb. Each object has keys:
  - verb (string): the target verb as it appears in the draft.
  - correct_form (string): its correct form in this context.
  - rule (string): the name of the rule (e.g. "verbo + preposición fija").
  - why (string): why the rule applies here.
  - es_contrast (string): the contrast with Spanish (L1 interference).
  - alternatives (array of strings): admissible alternatives and their nuance; may be empty.
- lexical_clarifications (array of objects, in Spanish): one entry per vocabulary or collocation problem in the sentence; never collapse several into one entry. Return at most three entries; if there are more, keep the three most important. Use an empty array when there is nothing to clarify. Each object has keys:
  - term (string): the word or expression analyzed.
  - meaning (string): what the correct form means.
  - why_wrong (string): why the learner's choice does not fit.
  - alternatives (array of strings): admissible alternatives and their nuance; may be empty.
- grammar_explanations (array of objects, in Spanish): one entry per grammar rule behind a mistake in the sentence; never collapse several into one entry. Return at most three entries; if there are more, keep the three most important. Use an empty array when there is no grammar issue. Each object has keys:
  - rule_name (string): the name of the grammar rule.
  - explanation (string): the logic behind it (the why).
  - construction (string): how it is built (the pattern).
  - counterexample (string): the learner's sentence corrected, as a counterexample.
  - exception (string): when the rule does NOT apply.
  - es_contrast (string): the contrast with Spanish.
- error_patterns (array): the mistakes found. Use an empty array when there is no mistake. Each item is an object with keys code (one of the codes above), severity (one of the severities above) and note (a short explanation in Spanish).

Prioritize: list every mistake in error_patterns, but explain in the structured sections only the most important ones (at most three entries per category). Never repeat the same explanation across entries, and never pad a field with filler. Every structured entry must correspond to a real mistake. Every string above must carry real content. When a category does not apply, return an empty array for that category instead of inventing an entry or leaving a field blank.`

// PromptText returns the exact system and user messages used for the first
// sentence of the draft. It is exported so operators and the manual
// `cmd/llmcheck` runner can audit the prompt without calling the API.
func PromptText(req ports.ExtractRequest) (system, user string) {
	sentences := SplitSentences(req.DraftText)
	sentence := ""
	if len(sentences) > 0 {
		sentence = sentences[0]
	}
	p := buildSentencePrompt(req, 0, len(sentences), sentence)
	return p.System, p.User
}

// buildSentencePrompt constructs the anonymized prompt for one sentence of the
// learner's draft (4.1.2). The request fields are already anonymized upstream
// (A8); this function does not touch personal data.
func buildSentencePrompt(req ports.ExtractRequest, index, total int, sentence string) prompt {
	return prompt{
		System: fmt.Sprintf(systemPrompt, errorPatternCatalog(), joinValues(errorPatternSeverities)),
		User:   buildSentenceUserPrompt(req, index, total, sentence),
	}
}

// buildSentenceUserPrompt renders the source, the target sentence and the
// practice rules as the user message.
func buildSentenceUserPrompt(req ports.ExtractRequest, index, total int, sentence string) string {
	var b strings.Builder

	b.WriteString("Source text (Spanish):\n")
	b.WriteString(req.SourceText)
	fmt.Fprintf(&b, "\n\nLearner's English sentence %d of %d to correct:\n", index+1, total)
	b.WriteString(sentence)
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

// SplitSentences breaks a text into sentences on . ! ? boundaries, keeping the
// terminator and trimming surrounding whitespace. The adapter calls the model
// once per sentence, so the learner's draft is segmented deterministically and
// the fragments can never drift out of alignment.
func SplitSentences(text string) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}

	var sentences []string
	start := 0
	for i := 0; i < len(runes); i++ {
		if runes[i] != '.' && runes[i] != '!' && runes[i] != '?' {
			continue
		}
		end := i + 1
		for end < len(runes) && (runes[end] == '.' || runes[end] == '!' || runes[end] == '?') {
			end++
		}
		if sentence := strings.TrimSpace(string(runes[start:end])); sentence != "" {
			sentences = append(sentences, sentence)
		}
		start = end
		i = end - 1
	}
	if start < len(runes) {
		if tail := strings.TrimSpace(string(runes[start:])); tail != "" {
			sentences = append(sentences, tail)
		}
	}
	return sentences
}

// errorPatternCatalog renders the code taxonomy as a bullet list with meanings.
func errorPatternCatalog() string {
	var b strings.Builder
	for _, item := range errorPatternDescriptions {
		fmt.Fprintf(&b, "\n- %s: %s", item.code, item.description)
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
