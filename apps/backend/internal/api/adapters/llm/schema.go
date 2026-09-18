package llm

import (
	"github.com/openai/openai-go"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// fragmentResponseSchemaName is the name OpenAI requires for a response_format
// JSON schema. It is stable so audits and logs can reference it.
const fragmentResponseSchemaName = "fragment_analysis"

// fragmentResponseSchemaDescription documents the schema for the provider.
const fragmentResponseSchemaDescription = "Fragment-by-fragment analysis of a Spanish source and an English learner draft."

// fragmentEnvelopeKey is the root property that carries the fragment list.
// Structured Outputs requires the root schema to be an object, so Fragment[] is
// wrapped as {"fragments": [...]} (4.2.1); the adapter unwraps it.
const fragmentEnvelopeKey = "fragments"

// fragmentEnvelope is the typed form of the response_format object.
type fragmentEnvelope struct {
	Fragments []analysis.Fragment `json:"fragments"`
}

// fragmentResponseFormat builds the strict Structured Outputs response format
// for Fragment[] (4.2.1). The provider then returns JSON that conforms to the
// schema; the adapter still validates it locally before persisting (4.2.2).
func fragmentResponseFormat() openai.ChatCompletionNewParamsResponseFormatUnion {
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        fragmentResponseSchemaName,
				Description: openai.String(fragmentResponseSchemaDescription),
				Schema:      fragmentSchema(),
				Strict:      openai.Bool(true),
			},
		},
	}
}

// fragmentSchema returns the JSON Schema of PRODUCT_DOMAIN §5.2 used with
// OpenAI Structured Outputs (4.2.1). It mirrors the OpenAPI Fragment contract,
// wrapped in a root object because Structured Outputs rejects a top-level
// array. The error pattern code/severity enums are derived from the domain
// constants so the schema can never drift from the taxonomy.
//
// Strict mode requires every property to be listed in "required" and
// "additionalProperties" to be false; note is therefore required (it may be an
// empty string) even though the wire contract keeps it optional.
func fragmentSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			fragmentEnvelopeKey: map[string]any{
				"type":  "array",
				"items": fragmentItemSchema(),
			},
		},
		"required":             []string{fragmentEnvelopeKey},
		"additionalProperties": false,
	}
}

// fragmentItemSchema is the schema of a single Fragment (PRODUCT_DOMAIN §5.2).
func fragmentItemSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source_es":             map[string]any{"type": "string"},
			"user_draft":            map[string]any{"type": "string"},
			"correction":            map[string]any{"type": "string"},
			"target_verb_review":    map[string]any{"type": "string"},
			"lexical_clarification": map[string]any{"type": "string"},
			"grammar_explanation":   map[string]any{"type": "string"},
			"error_patterns": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code":     map[string]any{"type": "string", "enum": errorPatternCodeEnum()},
						"severity": map[string]any{"type": "string", "enum": severityEnum()},
						"note":     map[string]any{"type": "string"},
					},
					"required":             []string{"code", "severity", "note"},
					"additionalProperties": false,
				},
			},
		},
		"required": []string{
			"source_es",
			"user_draft",
			"correction",
			"target_verb_review",
			"lexical_clarification",
			"grammar_explanation",
			"error_patterns",
		},
		"additionalProperties": false,
	}
}

// errorPatternCodeEnum lists the taxonomy codes in schema (string) form.
func errorPatternCodeEnum() []string {
	codes := make([]string, len(errorPatternDescriptions))
	for i, item := range errorPatternDescriptions {
		codes[i] = string(item.code)
	}
	return codes
}

// severityEnum lists the known severities in schema (string) form.
func severityEnum() []string {
	severities := make([]string, len(errorPatternSeverities))
	for i, severity := range errorPatternSeverities {
		severities[i] = string(severity)
	}
	return severities
}
