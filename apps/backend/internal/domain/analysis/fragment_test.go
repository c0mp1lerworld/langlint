package analysis_test

import (
	"encoding/json"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

func TestFragment_MarshalJSON_UsesSnakeCase(t *testing.T) {
	fragment := analysis.Fragment{
		SourceES:   "El gato duerme.",
		UserDraft:  "The cat sleep.",
		Correction: "The cat sleeps.",
		TargetVerbReview: analysis.TargetVerbReview{
			Verb:         "sleep",
			CorrectForm:  "sleeps",
			Rule:         "tercera persona singular",
			Why:          "el sujeto es singular",
			ESContrast:   "en español no cambia la forma",
			Alternatives: []string{"sleeps"},
		},
		LexicalClarification: analysis.LexicalClarification{
			Term:         "cat",
			Meaning:      "gato",
			WhyWrong:     "",
			Alternatives: nil,
		},
		GrammarExplanation: analysis.GrammarExplanation{
			RuleName:       "tercera persona singular",
			Explanation:    "el verbo añade -s",
			Construction:   "verbo + -s",
			Counterexample: "sleep -> sleeps",
			Exception:      "irregulares",
			ESContrast:     "no aplica en español",
		},
		ErrorPatterns: []domain.ErrorPattern{},
	}

	raw, err := json.Marshal(fragment)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{
		"source_es", "user_draft", "correction", "target_verb_review",
		"lexical_clarification", "grammar_explanation", "error_patterns",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 7 {
		t.Fatalf("payload %s has %d keys, want exactly 7", raw, len(got))
	}

	review, ok := got["target_verb_review"].(map[string]any)
	if !ok {
		t.Fatalf("target_verb_review is not an object: %s", raw)
	}
	for _, key := range []string{"verb", "correct_form", "rule", "why", "es_contrast", "alternatives"} {
		if _, ok := review[key]; !ok {
			t.Fatalf("target_verb_review missing key %q: %s", key, raw)
		}
	}
}
