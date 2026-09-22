package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

// studySessionSchemaName is the stable name OpenAI requires for the response
// format schema of a generated study session.
const studySessionSchemaName = "study_session"

// OpenAIStudySessionGenerator implements ports.StudySessionGenerator on top of
// the official OpenAI SDK (checklist 8.3.1). Like the extractor and the
// questioner, it is injected through the port so the provider stays an
// implementation detail (A1) and runs outside any transaction. Its input is the
// aggregated weakness profile — codes, severities and frequencies — so no raw
// learner text (and therefore no PII) ever leaves the process (A8).
type OpenAIStudySessionGenerator struct {
	client openai.Client
	model  string
}

var _ ports.StudySessionGenerator = (*OpenAIStudySessionGenerator)(nil)

// NewOpenAIStudySessionGenerator builds the adapter bound to a concrete model.
func NewOpenAIStudySessionGenerator(client openai.Client, model string) *OpenAIStudySessionGenerator {
	return &OpenAIStudySessionGenerator{client: client, model: model}
}

// Generate produces the session content — summarized theory, the learner's
// common traps and interactive exercises — from the weakness profile, using
// strict Structured Outputs (8.3.2). Any provider, transport, empty-response,
// truncation or schema violation is surfaced as a domain error (A5): a
// *domain.LLMOutputTruncatedError when the provider hit its output cap and a
// *domain.LLMUnavailableError otherwise, so no raw provider detail leaks.
func (g *OpenAIStudySessionGenerator) Generate(ctx context.Context, req ports.StudySessionRequest) (ports.StudySessionContent, error) {
	system, user := studySessionPrompt(req.Profile)

	completion, err := g.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: g.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		ResponseFormat: strictResponseFormat(studySessionSchemaName, studySessionSchema()),
		Temperature:    openai.Float(extractionTemperature),
	})
	if err != nil {
		return ports.StudySessionContent{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if len(completion.Choices) == 0 {
		return ports.StudySessionContent{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}
	if completion.Choices[0].FinishReason == finishReasonLength {
		return ports.StudySessionContent{}, &domain.LLMOutputTruncatedError{Message: "llm output truncated"}
	}

	var raw sessionContentRaw
	if err := json.Unmarshal([]byte(strings.TrimSpace(completion.Choices[0].Message.Content)), &raw); err != nil {
		return ports.StudySessionContent{}, &domain.LLMUnavailableError{Message: "llm unavailable"}
	}

	content, err := buildSessionContent(raw)
	if err != nil {
		return ports.StudySessionContent{}, err
	}
	return content, nil
}

// sessionContentRaw is the wire shape of the generated session content. It
// mirrors the strict schema; the domain value objects are rebuilt from it.
type sessionContentRaw struct {
	Theory string `json:"theory"`
	Traps  []struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"traps"`
	Exercises []struct {
		Kind   string `json:"kind"`
		Prompt string `json:"prompt"`
		Answer string `json:"answer"`
	} `json:"exercises"`
}

// buildSessionContent validates the raw output against the session contract
// (8.3.3) and rebuilds the domain value objects. Strict Structured Outputs
// already makes the provider conform, but the adapter never trusts the model: a
// missing theory, an unknown code or kind, or an empty trap/exercise list is
// reported as a provider failure (A5, A8).
func buildSessionContent(raw sessionContentRaw) (ports.StudySessionContent, error) {
	theory := strings.TrimSpace(raw.Theory)
	if theory == "" {
		return ports.StudySessionContent{}, invalidOutput()
	}

	traps := make([]tutor.Trap, 0, len(raw.Traps))
	for _, t := range raw.Traps {
		trap, err := tutor.NewTrap(domain.ErrorPatternCode(t.Code), t.Description)
		if err != nil {
			return ports.StudySessionContent{}, invalidOutput()
		}
		traps = append(traps, trap)
	}

	exercises := make([]tutor.Exercise, 0, len(raw.Exercises))
	for _, e := range raw.Exercises {
		exercise, err := tutor.NewExercise(tutor.ExerciseKind(e.Kind), e.Prompt, e.Answer)
		if err != nil {
			return ports.StudySessionContent{}, invalidOutput()
		}
		exercises = append(exercises, exercise)
	}

	if len(traps) == 0 || len(exercises) == 0 {
		return ports.StudySessionContent{}, invalidOutput()
	}

	return ports.StudySessionContent{Theory: theory, Traps: traps, Exercises: exercises}, nil
}

// studySessionSchema returns the strict JSON Schema of the session content
// (8.3.2). The code and kind enums are derived from the domain constants so the
// schema can never drift from the taxonomy.
func studySessionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"theory": map[string]any{"type": "string"},
			"traps": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code":        map[string]any{"type": "string", "enum": errorPatternCodeEnum()},
						"description": map[string]any{"type": "string"},
					},
					"required":             []string{"code", "description"},
					"additionalProperties": false,
				},
			},
			"exercises": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"kind":   map[string]any{"type": "string", "enum": []string{string(tutor.ExerciseKindOpen), string(tutor.ExerciseKindFill)}},
						"prompt": map[string]any{"type": "string"},
						"answer": map[string]any{"type": "string"},
					},
					"required":             []string{"kind", "prompt", "answer"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"theory", "traps", "exercises"},
		"additionalProperties": false,
	}
}

// studySessionSystemPrompt instructs the model to produce a theory summary plus
// exactly three learner traps and five personalized exercises (8.3.2).
const studySessionSystemPrompt = `You are a native English teacher building a personalized study session for a Spanish-speaking learner.
Based ONLY on the learner's recurring error patterns (weaknesses) listed by the user, produce:
- "theory": a concise summary in Spanish of the grammar topic behind the learner's most frequent weakness, naming the rule and explaining the why (3-5 sentences).
- "traps": EXACTLY three common traps the learner keeps falling into. Each is an object with "code" (one of the error pattern codes below) and "description" in Spanish explaining the specific mistake and why it keeps happening.
- "exercises": EXACTLY five interactive exercises personalized to those weaknesses. Each has "kind" ("fill" for a cloze sentence with a blank "___", or "open" for a question in Spanish asking the learner to explain the rule) and "prompt" (what the learner reads) and "answer" (the expected correct answer).
Never invent error pattern codes: use only the codes provided in the learner's weaknesses.
Respond with ONLY a JSON object of the shape {"theory": "...", "traps": [...], "exercises": [...]}. Do not wrap it in markdown code fences and do not add any prose.`

// studySessionPrompt renders the system and user messages from the profile. The
// user message lists only codes, severities and frequencies: no raw learner text
// is sent to the provider (A8).
func studySessionPrompt(profile tutor.WeaknessProfile) (system, user string) {
	return studySessionSystemPrompt, studySessionUserPrompt(profile)
}

// studySessionUserPrompt renders the aggregated weaknesses for the model.
func studySessionUserPrompt(profile tutor.WeaknessProfile) string {
	var b strings.Builder
	b.WriteString("The learner's recurring weaknesses (code, severity, frequency):")
	for _, entry := range profile.Entries {
		fmt.Fprintf(&b, "\n- %s (%s): %d occurrences", entry.Code, entry.Severity, entry.Count)
	}
	return b.String()
}
