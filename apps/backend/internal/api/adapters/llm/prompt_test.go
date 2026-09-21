package llm

import (
	"strings"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestSplitSentences_SplitsOnTerminators(t *testing.T) {
	got := SplitSentences("One. Two! Three? Four")
	want := []string{"One.", "Two!", "Three?", "Four"}
	if len(got) != len(want) {
		t.Fatalf("SplitSentences() = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitSentences()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitSentences_EmptyText_ReturnsNil(t *testing.T) {
	if got := SplitSentences("   "); got != nil {
		t.Fatalf("SplitSentences() = %q, want nil", got)
	}
}

func TestBuildSentencePrompt_IncludesSourceSentenceAndTargetRules(t *testing.T) {
	req := ports.ExtractRequest{
		PracticeID: domain.MustNewID(),
		SourceText: "El perro corre en el parque.",
		DraftText:  "The dog run in the park.",
		TargetRules: []practice.TargetRule{
			{Verb: "run", Tense: "past simple", Note: "irregular verb"},
		},
	}

	p := buildSentencePrompt(req, 0, 1, "The dog run in the park.")

	for _, want := range []string{
		req.SourceText,
		"The dog run in the park.",
		"1 of 1",
		"run",
		"past simple",
		"irregular verb",
	} {
		if !strings.Contains(p.User, want) {
			t.Fatalf("user prompt missing %q:\n%s", want, p.User)
		}
	}
}

func TestBuildSentencePrompt_NoTargetRules_RendersPlaceholder(t *testing.T) {
	p := buildSentencePrompt(ports.ExtractRequest{SourceText: "hola", DraftText: "hello"}, 0, 1, "hello")

	if !strings.Contains(p.User, "(none)") {
		t.Fatalf("user prompt missing empty-rules placeholder:\n%s", p.User)
	}
}

func TestBuildSentencePrompt_SystemPromptDescribesOutputContract(t *testing.T) {
	p := buildSentencePrompt(ports.ExtractRequest{}, 0, 1, "")

	for _, want := range []string{
		"JSON",
		"source_es",
		"user_draft",
		"correction",
		"target_verb_reviews",
		"lexical_clarifications",
		"grammar_explanations",
		"error_patterns",
	} {
		if !strings.Contains(p.System, want) {
			t.Fatalf("system prompt missing contract field %q:\n%s", want, p.System)
		}
	}
}

func TestBuildSentencePrompt_SystemPromptListsErrorTaxonomy(t *testing.T) {
	p := buildSentencePrompt(ports.ExtractRequest{}, 0, 1, "")

	for _, code := range []string{
		"infinitive_conjugation",
		"passive_voice_misuse",
		"idiom_literal_translation",
		"preposition_infinitive",
		"pronoun_possession",
		"false_friend",
		"lexical_choice",
		"word_order",
		"tense_agreement",
	} {
		if !strings.Contains(p.System, code) {
			t.Fatalf("system prompt missing error code %q:\n%s", code, p.System)
		}
	}

	for _, severity := range []string{"minor", "moderate", "critical"} {
		if !strings.Contains(p.System, severity) {
			t.Fatalf("system prompt missing severity %q:\n%s", severity, p.System)
		}
	}
}

func TestBuildSentencePrompt_SystemPromptRequiresOneFragmentPerSentence(t *testing.T) {
	p := buildSentencePrompt(ports.ExtractRequest{}, 0, 1, "")

	for _, want := range []string{
		"exactly one fragment",
		"never collapse several",
		"at most three entries",
		"Prioritize",
	} {
		if !strings.Contains(p.System, want) {
			t.Fatalf("system prompt missing completeness rule %q:\n%s", want, p.System)
		}
	}
}

func TestBuildSentencePrompt_RequiresVerbatimFragmentsAndSpanishExplanations(t *testing.T) {
	p := buildSentencePrompt(ports.ExtractRequest{}, 0, 1, "")

	if !strings.Contains(p.System, "verbatim") {
		t.Fatalf("system prompt must require verbatim fragments:\n%s", p.System)
	}
	if !strings.Contains(p.System, "Spanish") {
		t.Fatalf("system prompt must state the explanation language:\n%s", p.System)
	}
}

func TestPromptText_MatchesBuildSentencePrompt(t *testing.T) {
	req := ports.ExtractRequest{SourceText: "fuente", DraftText: "One. Two."}

	system, user := PromptText(req)
	p := buildSentencePrompt(req, 0, 2, "One.")

	if system != p.System || user != p.User {
		t.Fatal("PromptText() diverges from buildSentencePrompt()")
	}
}
