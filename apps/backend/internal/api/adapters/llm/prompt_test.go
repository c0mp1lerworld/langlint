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

func TestStripEllipsis_LeadingTrailingAndUnicode(t *testing.T) {
	cases := map[string]string{
		"...this continues.": "this continues.",
		"that continues...":  "that continues",
		"…this too…":         "this too",
		"normal sentence.":   "normal sentence.",
		"...":                "",
	}
	for in, want := range cases {
		if got := stripEllipsis(in); got != want {
			t.Errorf("stripEllipsis(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitSegments_StripsLeadingEllipsis(t *testing.T) {
	text := "...this will allow us to tie the commitments. In the end, I prefer not to lose hope."

	got := splitSegments(text)
	if len(got) != 2 {
		t.Fatalf("splitSegments() = %q, want 2 segments", got)
	}
	if strings.HasPrefix(got[0], "...") || got[0] == "..." {
		t.Fatalf("splitSegments()[0] = %q, want the leading ellipsis stripped", got[0])
	}
}

func TestSplitSegments_ShortDraft_SingleSegment(t *testing.T) {
	for _, text := range []string{"Por supuesto.", "Yes.", "No."} {
		got := splitSegments(text)
		if len(got) != 1 || got[0] != text {
			t.Errorf("splitSegments(%q) = %q, want a single segment [%q]", text, got, text)
		}
	}
}

func TestSplitRunOns_DoesNotDetachTransitionWord(t *testing.T) {
	sentence := "Then, while I arrange to sweep the backyard where the peasants swept in the afternoon, I understand the sadness that makes me weep, and I remember the secrets that I always kept with care."

	got := splitRunOns(sentence)
	if len(got) != 2 {
		t.Fatalf("splitRunOns() = %q, want 2 sub-clauses", got)
	}
	if !strings.HasPrefix(got[0], "Then, while ") {
		t.Fatalf("first sub-clause = %q, want it to keep \"Then,\" attached to the clause", got[0])
	}
	if !reconstructs(sentence, got) {
		t.Fatalf("splitRunOns() pieces do not reconstruct the original: %q", got)
	}
}

func TestSplitRunOns_ShortSentence_Unchanged(t *testing.T) {
	sentence := "The dog runs in the park."

	got := splitRunOns(sentence)
	if len(got) != 1 || got[0] != sentence {
		t.Fatalf("splitRunOns() = %q, want [%q]", got, sentence)
	}
}

func TestSplitRunOns_LongRunOn_SplitsAtCommaConjunctions(t *testing.T) {
	sentence := "I went to the market because I needed some milk, and I bought bread, and then I walked home while the sun was setting behind the buildings."

	got := splitRunOns(sentence)
	if len(got) != 3 {
		t.Fatalf("splitRunOns() = %q, want 3 sub-clauses", got)
	}
	if !reconstructs(sentence, got) {
		t.Fatalf("splitRunOns() pieces do not reconstruct the original: %q", got)
	}
	if !strings.HasPrefix(got[1], "and ") {
		t.Fatalf("second sub-clause = %q, want it to start with the connector", got[1])
	}
}

func TestSplitRunOns_LongSentenceWithoutConnectors_Unchanged(t *testing.T) {
	sentence := "The quick brown fox jumps over the lazy dog while everyone in the whole town watches the scene from the square with great attention and delight."

	got := splitRunOns(sentence)
	// The only boundary candidate is ", and" absent from this sentence, so it is
	// kept as a single fragment even though it is long.
	for _, part := range got {
		if part == "" {
			t.Fatal("splitRunOns() returned an empty piece")
		}
	}
	if len(got) != 1 {
		t.Fatalf("splitRunOns() = %q, want the original sentence (no clause boundary)", got)
	}
}

func TestSplitSegments_MixesSentencesAndRunOns(t *testing.T) {
	text := "Short one. I went to the market because I needed some milk, and I bought bread, and then I walked home while the sun was setting behind the buildings."

	got := splitSegments(text)
	if len(got) != 4 {
		t.Fatalf("splitSegments() = %q, want 4 segments", got)
	}
	if got[0] != "Short one." {
		t.Fatalf("splitSegments()[0] = %q, want %q", got[0], "Short one.")
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
