package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

func newQuestionerForHandler(t *testing.T, handler http.HandlerFunc) *OpenAITutorQuestioner {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := openai.NewClient(
		option.WithBaseURL(server.URL),
		option.WithAPIKey("test"),
		option.WithMaxRetries(0),
	)
	return NewOpenAITutorQuestioner(client, "gpt-4o-mini")
}

func TestOpenAITutorQuestioner_Question_ValidResponse_ReturnsQuestion(t *testing.T) {
	questioner := newQuestionerForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"kind":"fill","prompt":"I bet ___ the races."}`)
	})

	got, err := questioner.Question(context.Background(), ports.QuestionRequest{
		SourceText:  "Ayer aposté en las carreras.",
		UserDraft:   "Yesterday I bet in the races.",
		Correction:  "Yesterday I bet on the races.",
		TargetRules: nil,
		ErrorPatterns: []domain.ErrorPattern{{
			Code:     domain.ErrorPatternCodePrepositionInfinitive,
			Severity: domain.ErrorPatternSeverityModerate,
		}},
	})
	if err != nil {
		t.Fatalf("Question() error = %v", err)
	}
	if got.Kind != analysis.QuizKindFill || got.Prompt != "I bet ___ the races." {
		t.Fatalf("Question() = %+v", got)
	}
}

func TestOpenAITutorQuestioner_Question_RequestsStrictSchema(t *testing.T) {
	captured := make(chan map[string]any, 1)
	questioner := newQuestionerForHandler(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		captured <- body
		writeChatCompletion(t, w, `{"kind":"open","prompt":"¿Por qué?"}`)
	})

	if _, err := questioner.Question(context.Background(), ports.QuestionRequest{}); err != nil {
		t.Fatalf("Question() error = %v", err)
	}

	body := <-captured
	responseFormat, ok := body["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format = %v, want object", body["response_format"])
	}
	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok || jsonSchema["strict"] != true || jsonSchema["name"] != quizQuestionSchemaName {
		t.Fatalf("json_schema = %v", responseFormat["json_schema"])
	}
}

func TestOpenAITutorQuestioner_Question_InvalidKind_ReturnsLLMUnavailable(t *testing.T) {
	questioner := newQuestionerForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"kind":"mcq","prompt":"x"}`)
	})

	_, err := questioner.Question(context.Background(), ports.QuestionRequest{})
	assertLLMUnavailable(t, err)
}

func TestOpenAITutorQuestioner_Evaluate_ValidResponse_ReturnsEvaluation(t *testing.T) {
	questioner := newQuestionerForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"correct":true,"feedback":"¡Correcto!","follow_up":"¿Por qué no 'in'?"}`)
	})

	got, err := questioner.Evaluate(context.Background(), ports.EvaluateRequest{
		Question: "I bet ___ the races.",
		Answer:   "on",
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !got.Correct || got.Feedback != "¡Correcto!" || got.FollowUp != "¿Por qué no 'in'?" {
		t.Fatalf("Evaluate() = %+v", got)
	}
}

func TestOpenAITutorQuestioner_Evaluate_ProviderError_ReturnsLLMUnavailable(t *testing.T) {
	questioner := newQuestionerForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	})

	_, err := questioner.Evaluate(context.Background(), ports.EvaluateRequest{})
	assertLLMUnavailable(t, err)
}

func TestOpenAITutorQuestioner_Evaluate_EmptyFeedback_ReturnsLLMUnavailable(t *testing.T) {
	questioner := newQuestionerForHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		writeChatCompletion(t, w, `{"correct":false,"feedback":"   ","follow_up":""}`)
	})

	_, err := questioner.Evaluate(context.Background(), ports.EvaluateRequest{})
	assertLLMUnavailable(t, err)
}
