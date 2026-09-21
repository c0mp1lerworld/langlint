package llm

import (
	"encoding/json"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestFragmentSchema_MirrorsFragmentContract(t *testing.T) {
	raw, err := json.Marshal(fragmentSchema())
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Fatalf("schema.type = %v, want object", schema["type"])
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("schema.additionalProperties = %v, want false", schema["additionalProperties"])
	}
	assertStringList(t, schema["required"], []string{fragmentEnvelopeKey})

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema.properties = %v, want object", schema["properties"])
	}
	fragments, ok := properties[fragmentEnvelopeKey].(map[string]any)
	if !ok {
		t.Fatalf("properties.fragments = %v, want object", properties[fragmentEnvelopeKey])
	}
	if fragments["type"] != "array" {
		t.Fatalf("fragments.type = %v, want array", fragments["type"])
	}

	items, ok := fragments["items"].(map[string]any)
	if !ok {
		t.Fatalf("fragments.items = %v, want object", fragments["items"])
	}
	if items["additionalProperties"] != false {
		t.Fatalf("items.additionalProperties = %v, want false", items["additionalProperties"])
	}
	assertStringList(t, items["required"], []string{
		"source_es",
		"user_draft",
		"correction",
		"target_verb_reviews",
		"lexical_clarifications",
		"grammar_explanations",
		"error_patterns",
	})

	itemProperties, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("items.properties = %v, want object", items["properties"])
	}

	assertNestedArrayOfObjects(t, itemProperties["target_verb_reviews"], []string{
		"verb", "correct_form", "rule", "why", "es_contrast", "alternatives",
	})
	assertNestedArrayOfObjects(t, itemProperties["lexical_clarifications"], []string{
		"term", "meaning", "why_wrong", "alternatives",
	})
	assertNestedArrayOfObjects(t, itemProperties["grammar_explanations"], []string{
		"rule_name", "explanation", "construction", "counterexample", "exception", "es_contrast",
	})

	patterns, ok := itemProperties["error_patterns"].(map[string]any)
	if !ok {
		t.Fatalf("error_patterns = %v, want object", itemProperties["error_patterns"])
	}
	patternItems, ok := patterns["items"].(map[string]any)
	if !ok {
		t.Fatalf("error_patterns.items = %v, want object", patterns["items"])
	}
	if patternItems["additionalProperties"] != false {
		t.Fatalf("error_patterns.items.additionalProperties = %v, want false", patternItems["additionalProperties"])
	}
	assertStringList(t, patternItems["required"], []string{"code", "severity", "note"})

	patternProperties, ok := patternItems["properties"].(map[string]any)
	if !ok {
		t.Fatalf("error_patterns.items.properties = %v, want object", patternItems["properties"])
	}
	code, _ := patternProperties["code"].(map[string]any)
	assertStringList(t, code["enum"], []string{
		string(domain.ErrorPatternCodeInfinitiveConjugation),
		string(domain.ErrorPatternCodePassiveVoiceMisuse),
		string(domain.ErrorPatternCodeIdiomLiteralTranslation),
		string(domain.ErrorPatternCodePrepositionInfinitive),
		string(domain.ErrorPatternCodePronounPossession),
		string(domain.ErrorPatternCodeFalseFriend),
		string(domain.ErrorPatternCodeLexicalChoice),
		string(domain.ErrorPatternCodeWordOrder),
		string(domain.ErrorPatternCodeTenseAgreement),
	})

	severity, _ := patternProperties["severity"].(map[string]any)
	assertStringList(t, severity["enum"], []string{
		string(domain.ErrorPatternSeverityMinor),
		string(domain.ErrorPatternSeverityModerate),
		string(domain.ErrorPatternSeverityCritical),
	})
}

func TestFragmentResponseFormat_IsStrictJSONSchema(t *testing.T) {
	raw, err := json.Marshal(fragmentResponseFormat())
	if err != nil {
		t.Fatalf("marshal response format: %v", err)
	}

	var responseFormat map[string]any
	if err := json.Unmarshal(raw, &responseFormat); err != nil {
		t.Fatalf("unmarshal response format: %v", err)
	}
	if responseFormat["type"] != "json_schema" {
		t.Fatalf("response_format.type = %v, want json_schema", responseFormat["type"])
	}

	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("json_schema = %v, want object", responseFormat["json_schema"])
	}
	if jsonSchema["name"] != fragmentResponseSchemaName {
		t.Fatalf("json_schema.name = %v, want %s", jsonSchema["name"], fragmentResponseSchemaName)
	}
	if jsonSchema["strict"] != true {
		t.Fatalf("json_schema.strict = %v, want true", jsonSchema["strict"])
	}
	if _, ok := jsonSchema["schema"].(map[string]any); !ok {
		t.Fatalf("json_schema.schema = %v, want object", jsonSchema["schema"])
	}
}

// assertNestedArrayOfObjects checks a property is an array whose items are
// strict objects with exactly the expected required keys.
func assertNestedArrayOfObjects(t *testing.T, got any, wantRequired []string) {
	t.Helper()
	arr, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("nested array = %v (%T), want object", got, got)
	}
	if arr["type"] != "array" {
		t.Fatalf("nested array.type = %v, want array", arr["type"])
	}
	assertNestedObject(t, arr["items"], wantRequired)
}

// assertNestedObject checks an object property is strict (additionalProperties
// false) and has exactly the expected required keys.
func assertNestedObject(t *testing.T, got any, wantRequired []string) {
	t.Helper()
	obj, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("nested schema = %v (%T), want object", got, got)
	}
	if obj["type"] != "object" {
		t.Fatalf("nested schema.type = %v, want object", obj["type"])
	}
	if obj["additionalProperties"] != false {
		t.Fatalf("nested schema.additionalProperties = %v, want false", obj["additionalProperties"])
	}
	assertStringList(t, obj["required"], wantRequired)
}

func assertStringList(t *testing.T, got any, want []string) {
	t.Helper()
	values, ok := got.([]any)
	if !ok {
		t.Fatalf("list = %v (%T), want array", got, got)
	}
	if len(values) != len(want) {
		t.Fatalf("list = %v, want %v", values, want)
	}
	for i, value := range values {
		if value != want[i] {
			t.Fatalf("list[%d] = %v, want %v", i, value, want[i])
		}
	}
}
