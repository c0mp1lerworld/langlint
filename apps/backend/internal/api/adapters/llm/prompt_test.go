package llm

import (
	"strings"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestBuildPrompt_IncludesSourceDraftAndTargetRules(t *testing.T) {
	req := ports.ExtractRequest{
		PracticeID: domain.MustNewID(),
		SourceText: "El perro corre en el parque.",
		DraftText:  "The dog run in the park.",
		TargetRules: []practice.TargetRule{
			{Verb: "run", Tense: "past simple", Note: "irregular verb"},
		},
	}

	p := buildPrompt(req)

	for _, want := range []string{
		req.SourceText,
		req.DraftText,
		"run",
		"past simple",
		"irregular verb",
	} {
		if !strings.Contains(p.User, want) {
			t.Fatalf("user prompt missing %q:\n%s", want, p.User)
		}
	}
}

func TestBuildPrompt_NoTargetRules_RendersPlaceholder(t *testing.T) {
	p := buildPrompt(ports.ExtractRequest{SourceText: "hola", DraftText: "hello"})

	if !strings.Contains(p.User, "(none)") {
		t.Fatalf("user prompt missing empty-rules placeholder:\n%s", p.User)
	}
}

func TestBuildPrompt_SystemPromptDescribesOutputContract(t *testing.T) {
	p := buildPrompt(ports.ExtractRequest{})

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

func TestBuildPrompt_SystemPromptListsErrorTaxonomy(t *testing.T) {
	p := buildPrompt(ports.ExtractRequest{})

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

func TestBuildPrompt_RequiresVerbatimFragmentsAndSpanishExplanations(t *testing.T) {
	p := buildPrompt(ports.ExtractRequest{})

	if !strings.Contains(p.System, "verbatim") {
		t.Fatalf("system prompt must require verbatim fragments:\n%s", p.System)
	}
	if !strings.Contains(p.System, "Spanish") {
		t.Fatalf("system prompt must state the explanation language:\n%s", p.System)
	}
}

func TestPromptText_MatchesBuildPrompt(t *testing.T) {
	req := ports.ExtractRequest{SourceText: "fuente", DraftText: "draft"}

	system, user := PromptText(req)
	p := buildPrompt(req)

	if system != p.System || user != p.User {
		t.Fatal("PromptText() diverges from buildPrompt()")
	}
}
