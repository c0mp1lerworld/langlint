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

// Model returns the LLM model identifier used by the adapter.
func (e *OpenAIExtractor) Model() string { return e.model }

// ModelVersion returns the LLM model version used by the adapter.
func (e *OpenAIExtractor) ModelVersion() string { return e.modelVersion }

// Extract sends the anonymized practice to the LLM and returns the parsed
// fragments. Any provider, transport, empty-response or JSON parsing failure is
// surfaced as a *domain.LLMUnavailableError with a generic message, so no raw
// provider error leaks upstream (A5, A8). The call receives ctx and runs outside
// every transaction (PRODUCT_DOMAIN §4.7, AP6).
func (e *OpenAIExtractor) Extract(ctx context.Context, req ports.ExtractRequest) ([]analysis.Fragment, error) {
	p := buildPrompt(req)

	completion, err := e.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: e.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(p.System),
			openai.UserMessage(p.User),
		},
		ResponseFormat: fragmentResponseFormat(),
	})
	if err != nil {
		return nil, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if len(completion.Choices) == 0 {
		return nil, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}

	var fragments []analysis.Fragment
	if err := json.Unmarshal([]byte(strings.TrimSpace(completion.Choices[0].Message.Content)), &fragments); err != nil {
		return nil, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if err := validateFragments(fragments); err != nil {
		return nil, err
	}

	return fragments, nil
}
