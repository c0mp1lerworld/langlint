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
			"target_verb_review":    targetVerbReviewSchema(),
			"lexical_clarification": lexicalClarificationSchema(),
			"grammar_explanation":   grammarExplanationSchema(),
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

// stringProp is a JSON Schema string property.
func stringProp() map[string]any { return map[string]any{"type": "string"} }

// stringArrayProp is a JSON Schema array-of-strings property.
func stringArrayProp() map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
}

// targetVerbReviewSchema mirrors TargetVerbReview: the rule, the why, the
// Spanish contrast and the alternatives (4.2.1, Beyond Correction §1.2).
func targetVerbReviewSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"verb":         stringProp(),
			"correct_form": stringProp(),
			"rule":         stringProp(),
			"why":          stringProp(),
			"es_contrast":  stringProp(),
			"alternatives": stringArrayProp(),
		},
		"required":             []string{"verb", "correct_form", "rule", "why", "es_contrast", "alternatives"},
		"additionalProperties": false,
	}
}

// lexicalClarificationSchema mirrors LexicalClarification.
func lexicalClarificationSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"term":         stringProp(),
			"meaning":      stringProp(),
			"why_wrong":    stringProp(),
			"alternatives": stringArrayProp(),
		},
		"required":             []string{"term", "meaning", "why_wrong", "alternatives"},
		"additionalProperties": false,
	}
}

// grammarExplanationSchema mirrors GrammarExplanation: name, why, construction,
// counterexample, exceptions and the Spanish contrast.
func grammarExplanationSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"rule_name":      stringProp(),
			"explanation":    stringProp(),
			"construction":   stringProp(),
			"counterexample": stringProp(),
			"exception":      stringProp(),
			"es_contrast":    stringProp(),
		},
		"required":             []string{"rule_name", "explanation", "construction", "counterexample", "exception", "es_contrast"},
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
