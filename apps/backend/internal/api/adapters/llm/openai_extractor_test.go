package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// validFragmentContent is a single well-formed fragment for "El perro corre." /
// "The dog run." that every sentence-level call returns.
const validFragmentContent = `{"fragments":[{"source_es":"El perro corre.","user_draft":"The dog run.","correction":"The dog runs.","target_verb_reviews":[{"verb":"run","correct_form":"runs","rule":"tercera persona","why":"sujeto singular","es_contrast":"no cambia","alternatives":[]}],"lexical_clarifications":[],"grammar_explanations":[],"error_patterns":[]}]}`

func newExtractorForHandler(t *testing.T, handler http.HandlerFunc) *OpenAIExtractor {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := openai.NewClient(
		option.WithBaseURL(server.URL),
		option.WithAPIKey("test"),
		option.WithMaxRetries(0),
	)
	return NewOpenAIExtractor(client, "gpt-4o-mini", "2024-07-18")
}

func writeChatCompletion(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": 0,
		"model":   "gpt-4o-mini",
		"choices": []map[string]any{
			{
				"index":         0,
				"finish_reason": "stop",
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
			},
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("encode chat completion: %v", err)
	}
}

func assertLLMUnavailable(t *testing.T, err error) {
	t.Helper()
	var target *domain.LLMUnavailableError
	if !errors.As(err, &target) {
		t.Fatalf("Extract() error = %v, want *domain.LLMUnavailableError", err)
	}
}

func TestOpenAIExtractor_Extract_ValidResponse_ReturnsFragments(t *testing.T) {
	want := []analysis.Fragment{
		{
			SourceES:   "El perro corre.",
			UserDraft:  "The dog run.",
			Correction: "The dog runs.",
			TargetVerbReviews: []analysis.TargetVerbReview{{
				Verb:         "run",
				CorrectForm:  "runs",
				Rule:         "tercera persona singular",
				Why:          "el sujeto es singular",
				ESContrast:   "en español la forma no cambia",
				Alternatives: []string{"runs"},
			}},
			LexicalClarifications: []analysis.LexicalClarification{{
				Term:     "run",
				Meaning:  "correr",
				WhyWrong: "falta la -s de tercera persona",
			}},
			GrammarExplanations: []analysis.GrammarExplanation{{
				RuleName:       "tercera persona singular",
				Explanation:    "el verbo añade -s",
				Construction:   "verbo + -s",
				Counterexample: "run -> runs",
				Exception:      "verbos irregulares",
				ESContrast:     "no aplica en español",
			}},
			ErrorPatterns: []domain.ErrorPattern{
				{
					Code:     domain.ErrorPatternCodeInfinitiveConjugation,
					Severity: domain.ErrorPatternSeverityModerate,
					Note:     "missing third person -s",
				},
			},
		},
	}
	content, err := json.Marshal(map[string]any{fragmentEnvelopeKey: want})
	if err != nil {
		t.Fatalf("marshal fragments: %v", err)
	}

	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
		writeChatCompletion(t, w, string(content))
	})

	got, err := extractor.Extract(context.Background(), ports.ExtractRequest{
		PracticeID:  domain.MustNewID(),
		SourceText:  "El perro corre.",
		DraftText:   "The dog run.",
		TargetRules: []practice.TargetRule{{Verb: "run"}},
	})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(fragments) = %d, want 1", len(got))
	}
	if got[0].Correction != want[0].Correction {
		t.Fatalf("Correction = %q, want %q", got[0].Correction, want[0].Correction)
	}
	if !reflect.DeepEqual(got[0].TargetVerbReviews, want[0].TargetVerbReviews) {
		t.Fatalf("TargetVerbReviews = %+v, want %+v", got[0].TargetVerbReviews, want[0].TargetVerbReviews)
	}
	if !reflect.DeepEqual(got[0].GrammarExplanations, want[0].GrammarExplanations) {
		t.Fatalf("GrammarExplanations = %+v, want %+v", got[0].GrammarExplanations, want[0].GrammarExplanations)
	}
	if len(got[0].ErrorPatterns) != 1 || got[0].ErrorPatterns[0] != want[0].ErrorPatterns[0] {
		t.Fatalf("ErrorPatterns = %v, want %v", got[0].ErrorPatterns, want[0].ErrorPatterns)
	}
}

func TestOpenAIExtractor_Extract_MultipleSentences_OneCallPerSentence(t *testing.T) {
	calls := 0
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeChatCompletion(t, w, validFragmentContent)
	})

	got, err := extractor.Extract(context.Background(), ports.ExtractRequest{
		SourceText: "El perro corre. El gato duerme.",
		DraftText:  "The dog run. The cat sleep.",
	})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("provider calls = %d, want 2", calls)
	}
	if len(got) != 2 {
		t.Fatalf("len(fragments) = %d, want 2", len(got))
	}
	for i, want := range []string{"The dog run.", "The cat sleep."} {
		if got[i].UserDraft != want {
			t.Fatalf("fragments[%d].UserDraft = %q, want %q", i, got[i].UserDraft, want)
		}
	}
}

// TestOpenAIExtractor_Extract_LongRunOn_OneCallPerSubClause asserts a long
// run-on is subdivided and corrected sub-clause by sub-clause (BUG-001).
func TestOpenAIExtractor_Extract_LongRunOn_OneCallPerSubClause(t *testing.T) {
	calls := 0
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeChatCompletion(t, w, validFragmentContent)
	})

	draft := "I went to the market because I needed some milk, and I bought bread, and then I walked home while the sun was setting behind the buildings."
	got, err := extractor.Extract(context.Background(), ports.ExtractRequest{
		SourceText: "Fui al mercado porque necesitaba leche, y compré pan, y luego caminé a casa.",
		DraftText:  draft,
	})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if calls != 3 {
		t.Fatalf("provider calls = %d, want 3 (one per sub-clause)", calls)
	}
	if len(got) != 3 {
		t.Fatalf("len(fragments) = %d, want 3", len(got))
	}
}

func TestOpenAIExtractor_Extract_EmptyDraft_ReturnsNoFragments(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("provider must not be called for an empty draft")
		writeChatCompletion(t, w, validFragmentContent)
	})

	got, err := extractor.Extract(context.Background(), ports.ExtractRequest{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(fragments) = %d, want 0", len(got))
	}
}

func TestOpenAIExtractor_Extract_SendsAnonymizedPromptAndModel(t *testing.T) {
	captured := make(chan map[string]any, 1)
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		captured <- body
		writeChatCompletion(t, w, validFragmentContent)
	})

	if _, err := extractor.Extract(context.Background(), ports.ExtractRequest{
		SourceText:  "El perro corre.",
		DraftText:   "The dog run.",
		TargetRules: []practice.TargetRule{{Verb: "run"}},
	}); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	body := <-captured
	if body["model"] != "gpt-4o-mini" {
		t.Fatalf("model = %v, want gpt-4o-mini", body["model"])
	}

	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want 2 messages", body["messages"])
	}
	userMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("user message = %v, want object", messages[1])
	}
	userContent, _ := userMessage["content"].(string)
	for _, want := range []string{"El perro corre.", "The dog run.", "run", "1 of 1"} {
		if !strings.Contains(userContent, want) {
			t.Fatalf("user message missing %q:\n%s", want, userContent)
		}
	}
}

func TestOpenAIExtractor_Extract_ProviderError_ReturnsLLMUnavailableError(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom","type":"server_error"}}`))
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."})
	assertLLMUnavailable(t, err)
}

func TestOpenAIExtractor_Extract_InvalidJSON_ReturnsLLMUnavailableError(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, "not a json array")
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."})
	assertLLMUnavailable(t, err)
}

// TestOpenAIExtractor_Extract_TruncatedOutput_ReturnsLLMOutputTruncatedError
// asserts finish_reason=length is surfaced as a specific domain error instead of
// the generic llm_unavailable (BUG-002).
func TestOpenAIExtractor_Extract_TruncatedOutput_ReturnsLLMOutputTruncatedError(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 0,
			"model":   "gpt-4o-mini",
			"choices": []map[string]any{
				{
					"index":         0,
					"finish_reason": "length",
					"message":       map[string]any{"role": "assistant", "content": `{"fragments":[`},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."})

	var target *domain.LLMOutputTruncatedError
	if !errors.As(err, &target) {
		t.Fatalf("Extract() error = %v, want *domain.LLMOutputTruncatedError", err)
	}
}

func TestOpenAIExtractor_Extract_NoChoices_ReturnsLLMUnavailableError(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 0,
			"model":   "gpt-4o-mini",
			"choices": []any{},
		})
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."})
	assertLLMUnavailable(t, err)
}

func TestOpenAIExtractor_Extract_WrongFragmentCount_ReturnsLLMUnavailableError(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"fragments":[]}`)
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."})
	assertLLMUnavailable(t, err)
}

func TestOpenAIExtractor_Extract_InvalidErrorCode_ReturnsLLMUnavailableError(t *testing.T) {
	content := `{"fragments":[{"source_es":"El perro corre.","user_draft":"The dog run.","correction":"The dog runs.","target_verb_reviews":[{"verb":"run","correct_form":"runs","rule":"tercera persona","why":"sujeto singular","es_contrast":"no cambia","alternatives":["runs"]}],"lexical_clarifications":[{"term":"run","meaning":"correr","why_wrong":"falta -s","alternatives":[]}],"grammar_explanations":[{"rule_name":"tercera persona","explanation":"añade -s","construction":"verbo + -s","counterexample":"run -> runs","exception":"irregulares","es_contrast":"no aplica"}],"error_patterns":[{"code":"invented_code","severity":"minor","note":"x"}]}]}`
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, content)
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."})
	assertLLMUnavailable(t, err)
}

func TestOpenAIExtractor_Extract_RequestsStrictFragmentSchema(t *testing.T) {
	captured := make(chan map[string]any, 1)
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		captured <- body
		writeChatCompletion(t, w, validFragmentContent)
	})

	if _, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."}); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	body := <-captured
	responseFormat, ok := body["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format = %v, want object", body["response_format"])
	}
	if responseFormat["type"] != "json_schema" {
		t.Fatalf("response_format.type = %v, want json_schema", responseFormat["type"])
	}
	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("json_schema = %v, want object", responseFormat["json_schema"])
	}
	if jsonSchema["strict"] != true {
		t.Fatalf("json_schema.strict = %v, want true", jsonSchema["strict"])
	}
}

func TestOpenAIExtractor_Extract_RequestsDeterministicTemperature(t *testing.T) {
	captured := make(chan map[string]any, 1)
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		captured <- body
		writeChatCompletion(t, w, validFragmentContent)
	})

	if _, err := extractor.Extract(context.Background(), ports.ExtractRequest{DraftText: "The dog run."}); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	body := <-captured
	if body["temperature"] != float64(extractionTemperature) {
		t.Fatalf("temperature = %v, want %v", body["temperature"], extractionTemperature)
	}
}

func TestNewOpenAIExtractor_ModelMetadata(t *testing.T) {
	extractor := NewOpenAIExtractor(openai.Client{}, "gpt-4o-mini", "2024-07-18")

	if extractor.Model() != "gpt-4o-mini" {
		t.Fatalf("Model() = %q, want gpt-4o-mini", extractor.Model())
	}
	if extractor.ModelVersion() != "2024-07-18" {
		t.Fatalf("ModelVersion() = %q, want 2024-07-18", extractor.ModelVersion())
	}
}
