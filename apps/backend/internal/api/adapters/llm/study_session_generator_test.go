package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func newGeneratorForHandler(t *testing.T, handler http.HandlerFunc) *OpenAIStudySessionGenerator {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := openai.NewClient(
		option.WithBaseURL(server.URL),
		option.WithAPIKey("test"),
		option.WithMaxRetries(0),
	)
	return NewOpenAIStudySessionGenerator(client, "gpt-4o-mini")
}

func validProfile(t *testing.T) tutor.WeaknessProfile {
	t.Helper()
	entry, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodePrepositionInfinitive, domain.ErrorPatternSeverityModerate, 6, time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewWeaknessEntry() error = %v", err)
	}
	profile, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{entry})
	if err != nil {
		t.Fatalf("NewWeaknessProfile() error = %v", err)
	}
	return profile
}

func TestOpenAIStudySessionGenerator_Generate_ValidResponse_ReturnsContent(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{
			"theory": "Las preposiciones que rigen gerundio no se traducen literalmente.",
			"traps": [
				{"code": "preposition_infinitive", "description": "Usas 'to' antes del gerundio por calco del español."},
				{"code": "preposition_infinitive", "description": "Olvidas que algunos verbos exigen preposición fija."}
			],
			"exercises": [
				{"kind": "fill", "prompt": "I look forward ___ (see) you.", "answer": "to seeing"},
				{"kind": "open", "prompt": "¿Por qué no se dice 'bet in'?", "answer": "bet exige la preposición on"}
			]
		}`)
	})

	got, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.Theory == "" {
		t.Fatal("Theory is empty")
	}
	if len(got.Traps) != 2 {
		t.Fatalf("len(Traps) = %d, want 2", len(got.Traps))
	}
	if got.Traps[0].Code != domain.ErrorPatternCodePrepositionInfinitive {
		t.Fatalf("Traps[0].Code = %q, want preposition_infinitive", got.Traps[0].Code)
	}
	if len(got.Exercises) != 2 {
		t.Fatalf("len(Exercises) = %d, want 2", len(got.Exercises))
	}
	if got.Exercises[0].Kind != tutor.ExerciseKindFill || got.Exercises[0].Answer != "to seeing" {
		t.Fatalf("Exercises[0] = %+v", got.Exercises[0])
	}
}

func TestOpenAIStudySessionGenerator_Generate_RequestsStrictSchemaAndGreedy(t *testing.T) {
	captured := make(chan map[string]any, 1)
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		captured <- body
		writeChatCompletion(t, w, `{"theory":"x","traps":[{"code":"word_order","description":"y"}],"exercises":[{"kind":"fill","prompt":"p","answer":"a"}]}`)
	})

	if _, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	body := <-captured
	responseFormat, ok := body["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format = %v, want object", body["response_format"])
	}
	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok || jsonSchema["strict"] != true || jsonSchema["name"] != studySessionSchemaName {
		t.Fatalf("json_schema = %v", responseFormat["json_schema"])
	}
	if body["temperature"] != float64(0) {
		t.Fatalf("temperature = %v, want 0", body["temperature"])
	}
}

func TestOpenAIStudySessionGenerator_Generate_InvalidCode_ReturnsLLMUnavailable(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"theory":"x","traps":[{"code":"spelling","description":"y"}],"exercises":[{"kind":"fill","prompt":"p","answer":"a"}]}`)
	})

	_, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	assertLLMUnavailable(t, err)
}

func TestOpenAIStudySessionGenerator_Generate_InvalidKind_ReturnsLLMUnavailable(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"theory":"x","traps":[{"code":"word_order","description":"y"}],"exercises":[{"kind":"mcq","prompt":"p","answer":"a"}]}`)
	})

	_, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	assertLLMUnavailable(t, err)
}

func TestOpenAIStudySessionGenerator_Generate_EmptyTheory_ReturnsLLMUnavailable(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"theory":"   ","traps":[{"code":"word_order","description":"y"}],"exercises":[{"kind":"fill","prompt":"p","answer":"a"}]}`)
	})

	_, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	assertLLMUnavailable(t, err)
}

func TestOpenAIStudySessionGenerator_Generate_EmptyTraps_ReturnsLLMUnavailable(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"theory":"x","traps":[],"exercises":[{"kind":"fill","prompt":"p","answer":"a"}]}`)
	})

	_, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	assertLLMUnavailable(t, err)
}

func TestOpenAIStudySessionGenerator_Generate_Truncated_ReturnsLLMOutputTruncated(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"id":     "chatcmpl-test",
			"object": "chat.completion",
			"choices": []map[string]any{
				{"index": 0, "finish_reason": "length", "message": map[string]any{"role": "assistant", "content": "{\"theory\":"}},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode chat completion: %v", err)
		}
	})

	_, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	var target *domain.LLMOutputTruncatedError
	if !errors.As(err, &target) {
		t.Fatalf("Generate() error = %v, want *domain.LLMOutputTruncatedError", err)
	}
}

func TestOpenAIStudySessionGenerator_Generate_ProviderError_ReturnsLLMUnavailable(t *testing.T) {
	generator := newGeneratorForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	})

	_, err := generator.Generate(context.Background(), ports.StudySessionRequest{Profile: validProfile(t)})
	assertLLMUnavailable(t, err)
}

func TestStudySessionPrompt_ContainsOnlyAggregates(t *testing.T) {
	profile := validProfile(t)
	system, user := studySessionPrompt(profile)

	if system == "" || user == "" {
		t.Fatal("prompt must not be empty")
	}
	// The user message lists only codes, severities and frequencies: the enum
	// code and the count must appear, and there must be no free learner text.
	if !strings.Contains(user, string(domain.ErrorPatternCodePrepositionInfinitive)) {
		t.Fatalf("user prompt missing code, got %q", user)
	}
	if !strings.Contains(user, "6") {
		t.Fatalf("user prompt missing frequency, got %q", user)
	}
	if strings.Contains(user, "@") || strings.Contains(user, "[name]") {
		t.Fatalf("user prompt must carry no PII, got %q", user)
	}
}

func TestStudySessionSchema_RequiresTheoryTrapsExercises(t *testing.T) {
	schema := studySessionSchema()
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("schema.required = %v, want []string", schema["required"])
	}
	for _, key := range []string{"theory", "traps", "exercises"} {
		if !slices.Contains(required, key) {
			t.Fatalf("schema.required missing %q", key)
		}
	}
	if schema["additionalProperties"] != false {
		t.Fatal("schema.additionalProperties must be false")
	}
}
