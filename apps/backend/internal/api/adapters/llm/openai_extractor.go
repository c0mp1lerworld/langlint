// Package llm holds the LLM provider adapters of the API entry point (A2). The
// domain stays provider-agnostic through the ports.LLMExtractor interface (A1):
// no domain or service type imports the provider SDK.
package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// OpenAIExtractor implements ports.LLMExtractor on top of the official OpenAI
// SDK (PRODUCT_DOMAIN §6.1). It is injected through the port so the provider is
// an implementation detail (A1, item 4.1.3).
type OpenAIExtractor struct {
	client       openai.Client
	model        string
	modelVersion string
}

var _ ports.LLMExtractor = (*OpenAIExtractor)(nil)

// NewOpenAIExtractor builds the adapter bound to a concrete model. The model
// and its version are persisted with the Analysis for traceability
// (PRODUCT_DOMAIN §4.2.3).
func NewOpenAIExtractor(client openai.Client, model, modelVersion string) *OpenAIExtractor {
	return &OpenAIExtractor{
		client:       client,
		model:        model,
		modelVersion: modelVersion,
	}
}

// extractionTemperature pins the decoding to greedy sampling. The extraction is
// a structured, exhaustive task: a higher temperature made the model
// non-deterministically collapse several mistakes into a single explanation.
const extractionTemperature = 0.0

// Model returns the LLM model identifier used by the adapter.
func (e *OpenAIExtractor) Model() string { return e.model }

// ModelVersion returns the LLM model version used by the adapter.
func (e *OpenAIExtractor) ModelVersion() string { return e.modelVersion }

// Extract sends the anonymized practice to the LLM and returns the parsed
// fragments. It calls the model once per sentence of the learner's draft so the
// fragments cannot drift out of alignment when the source and the draft have
// different sentence counts; the draft segmentation is deterministic and owned
// here. Any provider, transport, empty-response or JSON parsing failure is
// surfaced as a *domain.LLMUnavailableError with a generic message, so no raw
// provider error leaks upstream (A5, A8). The calls receive ctx and run outside
// every transaction (PRODUCT_DOMAIN §4.7, AP6).
func (e *OpenAIExtractor) Extract(ctx context.Context, req ports.ExtractRequest) ([]analysis.Fragment, error) {
	sentences := SplitSentences(req.DraftText)
	if len(sentences) == 0 {
		return []analysis.Fragment{}, nil
	}

	fragments := make([]analysis.Fragment, 0, len(sentences))
	for i, sentence := range sentences {
		if err := ctx.Err(); err != nil {
			return nil, &domain.LLMUnavailableError{Message: "llm unavailable"}
		}
		fragment, err := e.extractSentence(ctx, req, i, len(sentences), sentence)
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, fragment)
	}
	if err := validateFragments(fragments); err != nil {
		return nil, err
	}

	return fragments, nil
}

// extractSentence corrects one sentence of the learner's draft. The model
// supplies the Spanish source span, the correction and the explanations; the
// user_draft is taken from our own deterministic split so the fragments always
// cover the draft exactly, in order (4.2.1/4.2.2).
func (e *OpenAIExtractor) extractSentence(ctx context.Context, req ports.ExtractRequest, index, total int, sentence string) (analysis.Fragment, error) {
	p := buildSentencePrompt(req, index, total, sentence)

	completion, err := e.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: e.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(p.System),
			openai.UserMessage(p.User),
		},
		ResponseFormat: fragmentResponseFormat(),
		Temperature:    openai.Float(extractionTemperature),
	})
	if err != nil {
		return analysis.Fragment{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if len(completion.Choices) == 0 {
		return analysis.Fragment{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}

	var envelope fragmentEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(completion.Choices[0].Message.Content)), &envelope); err != nil {
		return analysis.Fragment{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if len(envelope.Fragments) != 1 {
		return analysis.Fragment{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}

	fragment := envelope.Fragments[0]
	fragment.UserDraft = sentence
	return fragment, nil
}
