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
		"target_verb_review",
		"lexical_clarification",
		"grammar_explanation",
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
