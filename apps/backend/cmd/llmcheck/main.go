// Command llmcheck is the manual smoke runner of the LLM engine (Fase 4). It
// builds the real OpenAI adapter through the ports.LLMExtractor port, sends a
// sample (or file-provided) practice and prints the parsed Fragment[] as JSON.
//
// It is a developer tool, not part of the HTTP API. Run it from apps/backend:
//
//	go run ./cmd/llmcheck -show-prompt
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/llm"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "llmcheck:", err)
		os.Exit(1)
	}
}

// practiceInput is the JSON shape accepted by -input.
type practiceInput struct {
	SourceText  string                `json:"source_text"`
	DraftText   string                `json:"draft_text"`
	TargetRules []practice.TargetRule `json:"target_rules"`
}

var sampleInput = practiceInput{
	SourceText: "El año pasado, el perro de mi vecino se escapó y corrimos detrás de él por todo el parque.",
	DraftText:  "Last year, my neighbour's dog escaped and we run behind it across all the park.",
	TargetRules: []practice.TargetRule{
		{Verb: "run", Tense: "past simple", Note: "verbo irregular"},
	},
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("llmcheck", flag.ContinueOnError)
	flags.SetOutput(out)
	envFile := flags.String("env", ".env", "path to the .env file with the OpenAI settings")
	inputFile := flags.String("input", "", "path to a JSON practice input (default: built-in sample)")
	showPrompt := flags.Bool("show-prompt", false, "print the exact prompts sent to the model")
	timeout := flags.Duration("timeout", 60*time.Second, "overall LLM call timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if err := config.LoadDotEnv(*envFile); err != nil {
		return err
	}
	cfg, err := config.LoadOpenAIConfig()
	if err != nil {
		return err
	}

	input, err := loadInput(*inputFile)
	if err != nil {
		return err
	}

	req := ports.ExtractRequest{
		PracticeID:  domain.MustNewID(),
		SourceText:  input.SourceText,
		DraftText:   input.DraftText,
		TargetRules: input.TargetRules,
	}

	if *showPrompt {
		system, user := llm.PromptText(req)
		fmt.Fprintf(out, "===== SYSTEM PROMPT =====\n%s\n\n===== USER PROMPT =====\n%s\n\n", system, user)
	}

	options := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(llm.BaseURLOrDefault(cfg.BaseURL)),
	}
	extractor := llm.NewOpenAIExtractor(openai.NewClient(options...), cfg.Model, cfg.ModelVersion)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	fmt.Fprintf(out, "Calling %s (%s)...\n\n", cfg.Model, cfg.ModelVersion)
	fragments, err := extractor.Extract(ctx, req)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(fragments)
}

// loadInput reads the practice from path, or returns the built-in sample when
// path is empty.
func loadInput(path string) (practiceInput, error) {
	if path == "" {
		return sampleInput, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return practiceInput{}, err
	}

	var input practiceInput
	if err := json.Unmarshal(data, &input); err != nil {
		return practiceInput{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return input, nil
}
