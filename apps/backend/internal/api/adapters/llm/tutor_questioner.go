package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// Schema names required by OpenAI Structured Outputs. They are stable so audits
// can reference them.
const (
	quizQuestionSchemaName   = "quiz_question"
	quizEvaluationSchemaName = "quiz_evaluation"
)

// OpenAITutorQuestioner implements ports.TutorQuestioner on top of the official
// OpenAI SDK (PRODUCT_DOMAIN §1.2). Like the extractor, it is injected through
// the port so the provider is an implementation detail (A1) and runs outside any
// transaction.
type OpenAITutorQuestioner struct {
	client openai.Client
	model  string
}

var _ ports.TutorQuestioner = (*OpenAITutorQuestioner)(nil)

// NewOpenAITutorQuestioner builds the adapter bound to a concrete model.
func NewOpenAITutorQuestioner(client openai.Client, model string) *OpenAITutorQuestioner {
	return &OpenAITutorQuestioner{client: client, model: model}
}

// Question generates one active-practice question anchored to the fragment.
func (q *OpenAITutorQuestioner) Question(ctx context.Context, req ports.QuestionRequest) (analysis.QuizQuestion, error) {
	system, user := quizQuestionPrompt(req)

	var raw struct {
		Kind   string `json:"kind"`
		Prompt string `json:"prompt"`
	}
	if err := q.call(ctx, system, user, quizQuestionSchemaName, quizQuestionSchema(), &raw); err != nil {
		return analysis.QuizQuestion{}, err
	}

	question, err := analysis.NewQuizQuestion(analysis.QuizKind(raw.Kind), raw.Prompt)
	if err != nil {
		return analysis.QuizQuestion{}, invalidOutput()
	}
	return question, nil
}

// Evaluate grades the learner's answer and optionally adds a follow-up.
func (q *OpenAITutorQuestioner) Evaluate(ctx context.Context, req ports.EvaluateRequest) (analysis.QuizEvaluation, error) {
	system, user := quizEvaluationPrompt(req)

	var raw struct {
		Correct  bool   `json:"correct"`
		Feedback string `json:"feedback"`
		FollowUp string `json:"follow_up"`
	}
	if err := q.call(ctx, system, user, quizEvaluationSchemaName, quizEvaluationSchema(), &raw); err != nil {
		return analysis.QuizEvaluation{}, err
	}

	evaluation, err := analysis.NewQuizEvaluation(raw.Correct, raw.Feedback, raw.FollowUp)
	if err != nil {
		return analysis.QuizEvaluation{}, invalidOutput()
	}
	return evaluation, nil
}

// call sends a strict Structured Outputs request and unmarshals the JSON
// response into out. Any provider, transport or parsing failure is surfaced as
// a *domain.LLMUnavailableError with a generic message (A5, A8).
func (q *OpenAITutorQuestioner) call(ctx context.Context, system, user, schemaName string, schema map[string]any, out any) error {
	completion, err := q.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: q.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		ResponseFormat: strictResponseFormat(schemaName, schema),
	})
	if err != nil {
		return &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if len(completion.Choices) == 0 {
		return &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(completion.Choices[0].Message.Content)), out); err != nil {
		return &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	return nil
}

// strictResponseFormat builds a strict Structured Outputs response format.
func strictResponseFormat(name string, schema map[string]any) openai.ChatCompletionNewParamsResponseFormatUnion {
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   name,
				Schema: schema,
				Strict: openai.Bool(true),
			},
		},
	}
}

// quizQuestionSchema is the strict schema of a generated question.
func quizQuestionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind":   map[string]any{"type": "string", "enum": []string{string(analysis.QuizKindOpen), string(analysis.QuizKindFill)}},
			"prompt": map[string]any{"type": "string"},
		},
		"required":             []string{"kind", "prompt"},
		"additionalProperties": false,
	}
}

// quizEvaluationSchema is the strict schema of an evaluation.
func quizEvaluationSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"correct":   map[string]any{"type": "boolean"},
			"feedback":  map[string]any{"type": "string"},
			"follow_up": map[string]any{"type": "string"},
		},
		"required":             []string{"correct", "feedback", "follow_up"},
		"additionalProperties": false,
	}
}

// quizQuestionSystemPrompt turns the fragment's mistake into one focused
// production question (cloze for a concrete form, open for reasoning).
const quizQuestionSystemPrompt = `You are a native English teacher creating ONE short active-practice question for a Spanish-speaking learner, anchored to a specific mistake in their fragment.
Choose the best format and set "kind" accordingly:
- "fill": a cloze sentence in English with a blank ("___") where the learner must produce the corrected form. Use it for a concrete form (preposition, tense, verb form, article).
- "open": a question in Spanish asking the learner to explain WHY their draft is wrong and the rule behind it. Use it for lexical, idiomatic or word-order mistakes.
The "prompt" is what the learner reads: write the instruction in Spanish and, for a cloze, include the English sentence with the blank.
Never include the answer in the prompt. Keep it short and specific to the fragment's mistake.`

// quizEvaluationSystemPrompt grades an answer and optionally follows up.
const quizEvaluationSystemPrompt = `You are a native English teacher grading a Spanish-speaking learner's answer to an active-practice question.
Set "correct" to true only when the answer is right. Write short feedback in Spanish that names the rule and explains the why.
Optionally add ONE short Socratic follow-up question in Spanish to deepen understanding; use an empty string when no follow-up is useful.
Be encouraging and precise, and never reveal hidden reasoning.`

// quizQuestionPrompt renders the question-generation user message.
func quizQuestionPrompt(req ports.QuestionRequest) (system, user string) {
	var b strings.Builder
	fmt.Fprintf(&b, "Source (Spanish):\n%s\n\nLearner draft (English):\n%s\n\nCorrected sentence:\n%s",
		req.SourceText, req.UserDraft, req.Correction)
	fmt.Fprintf(&b, "\n\nErrors: %s\nTarget rules: %s", describePatterns(req.ErrorPatterns), describeTargetRules(req.TargetRules))
	return quizQuestionSystemPrompt, b.String()
}

// quizEvaluationPrompt renders the evaluation user message.
func quizEvaluationPrompt(req ports.EvaluateRequest) (system, user string) {
	var b strings.Builder
	fmt.Fprintf(&b, "Source (Spanish):\n%s\n\nLearner draft (English):\n%s\n\nCorrected sentence:\n%s",
		req.SourceText, req.UserDraft, req.Correction)
	fmt.Fprintf(&b, "\n\nErrors: %s", describePatterns(req.ErrorPatterns))
	fmt.Fprintf(&b, "\n\nQuestion:\n%s\n\nLearner answer:\n%s", req.Question, req.Answer)
	return quizEvaluationSystemPrompt, b.String()
}

// describePatterns renders the error patterns compactly for the prompt.
func describePatterns(patterns []domain.ErrorPattern) string {
	if len(patterns) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		parts = append(parts, fmt.Sprintf("%s (%s)", pattern.Code, pattern.Severity))
	}
	return strings.Join(parts, ", ")
}

// describeTargetRules renders the target rules compactly for the prompt.
func describeTargetRules(rules []practice.TargetRule) string {
	if len(rules) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(rules))
	for _, rule := range rules {
		parts = append(parts, rule.Verb)
	}
	return strings.Join(parts, ", ")
}
