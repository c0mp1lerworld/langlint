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
		TargetVerbReviews: []analysis.TargetVerbReview{{
			Verb:         "sleep",
			CorrectForm:  "sleeps",
			Rule:         "tercera persona singular",
			Why:          "el sujeto es singular",
			ESContrast:   "en español no cambia la forma",
			Alternatives: []string{"sleeps"},
		}},
		LexicalClarifications: []analysis.LexicalClarification{{
			Term:         "cat",
			Meaning:      "gato",
			WhyWrong:     "",
			Alternatives: nil,
		}},
		GrammarExplanations: []analysis.GrammarExplanation{{
			RuleName:       "tercera persona singular",
			Explanation:    "el verbo añade -s",
			Construction:   "verbo + -s",
			Counterexample: "sleep -> sleeps",
			Exception:      "irregulares",
			ESContrast:     "no aplica en español",
		}},
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
		"source_es", "user_draft", "correction", "target_verb_reviews",
		"lexical_clarifications", "grammar_explanations", "error_patterns",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 7 {
		t.Fatalf("payload %s has %d keys, want exactly 7", raw, len(got))
	}

	reviews, ok := got["target_verb_reviews"].([]any)
	if !ok {
		t.Fatalf("target_verb_reviews is not an array: %s", raw)
	}
	if len(reviews) != 1 {
		t.Fatalf("target_verb_reviews has %d items, want 1: %s", len(reviews), raw)
	}
	review, ok := reviews[0].(map[string]any)
	if !ok {
		t.Fatalf("target_verb_reviews[0] is not an object: %s", raw)
	}
	for _, key := range []string{"verb", "correct_form", "rule", "why", "es_contrast", "alternatives"} {
		if _, ok := review[key]; !ok {
			t.Fatalf("target_verb_reviews[0] missing key %q: %s", key, raw)
		}
	}
}

func TestFragment_MarshalJSON_KeepsMultipleExplanations(t *testing.T) {
	fragment := analysis.Fragment{
		SourceES:   "Ayer hablé con mi hermano y aposté.",
		UserDraft:  "Yesterday I talked with mi brother and I bet in horses.",
		Correction: "Yesterday I talked with my brother and I bet on horses.",
		TargetVerbReviews: []analysis.TargetVerbReview{
			{Verb: "bet", CorrectForm: "bet", Rule: "verbo + preposición fija", Why: "bet on", ESContrast: "apostar a"},
			{Verb: "talk", CorrectForm: "talked", Rule: "pasado simple regular", Why: "ayer", ESContrast: "hablé"},
		},
		LexicalClarifications: []analysis.LexicalClarification{
			{Term: "mi brother", Meaning: "my brother", WhyWrong: "posesivo"},
			{Term: "in horses", Meaning: "on horses", WhyWrong: "colocación"},
		},
		GrammarExplanations: []analysis.GrammarExplanation{
			{RuleName: "posesivos", Explanation: "my", Construction: "my + sustantivo", Counterexample: "mi -> my", Exception: "n/a", ESContrast: "mi"},
			{RuleName: "preposición", Explanation: "on", Construction: "bet on", Counterexample: "in -> on", Exception: "n/a", ESContrast: "a"},
		},
		ErrorPatterns: []domain.ErrorPattern{},
	}

	raw, err := json.Marshal(fragment)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got analysis.Fragment
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(got.TargetVerbReviews) != 2 || len(got.LexicalClarifications) != 2 || len(got.GrammarExplanations) != 2 {
		t.Fatalf("round-trip lost items: %+v", got)
	}
	if got.TargetVerbReviews[0].Verb != "bet" || got.TargetVerbReviews[1].Verb != "talk" {
		t.Fatalf("target verbs out of order: %+v", got.TargetVerbReviews)
	}
}
