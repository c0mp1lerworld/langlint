package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

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
			SourceES:             "El perro corre.",
			UserDraft:            "The dog run.",
			Correction:           "The dog runs.",
			TargetVerbReview:     "run (run/ran/run)",
			LexicalClarification: "correr = to run",
			GrammarExplanation:   "Third person singular takes -s.",
			ErrorPatterns: []domain.ErrorPattern{
				{
					Code:     domain.ErrorPatternCodeInfinitiveConjugation,
					Severity: domain.ErrorPatternSeverityModerate,
					Note:     "missing third person -s",
				},
			},
		},
	}
	content, err := json.Marshal(want)
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
	if len(got[0].ErrorPatterns) != 1 || got[0].ErrorPatterns[0] != want[0].ErrorPatterns[0] {
		t.Fatalf("ErrorPatterns = %v, want %v", got[0].ErrorPatterns, want[0].ErrorPatterns)
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
		writeChatCompletion(t, w, "[]")
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
	for _, want := range []string{"El perro corre.", "The dog run.", "run"} {
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

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{})
	assertLLMUnavailable(t, err)
}

func TestOpenAIExtractor_Extract_InvalidJSON_ReturnsLLMUnavailableError(t *testing.T) {
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, "not a json array")
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{})
	assertLLMUnavailable(t, err)
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

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{})
	assertLLMUnavailable(t, err)
}

func TestOpenAIExtractor_Extract_InvalidErrorCode_ReturnsLLMUnavailableError(t *testing.T) {
	content := `[{"source_es":"El perro corre.","user_draft":"The dog run.","correction":"The dog runs.","target_verb_review":"run","lexical_clarification":"correr = to run","grammar_explanation":"third person -s","error_patterns":[{"code":"invented_code","severity":"minor","note":"x"}]}]`
	extractor := newExtractorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, content)
	})

	_, err := extractor.Extract(context.Background(), ports.ExtractRequest{})
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
		writeChatCompletion(t, w, "[]")
	})

	if _, err := extractor.Extract(context.Background(), ports.ExtractRequest{}); err != nil {
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

func TestNewOpenAIExtractor_ModelMetadata(t *testing.T) {
	extractor := NewOpenAIExtractor(openai.Client{}, "gpt-4o-mini", "2024-07-18")

	if extractor.Model() != "gpt-4o-mini" {
		t.Fatalf("Model() = %q, want gpt-4o-mini", extractor.Model())
	}
	if extractor.ModelVersion() != "2024-07-18" {
		t.Fatalf("ModelVersion() = %q, want 2024-07-18", extractor.ModelVersion())
	}
}
