package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInput_NoPath_ReturnsSample(t *testing.T) {
	input, err := loadInput("")
	if err != nil {
		t.Fatalf("loadInput() error = %v", err)
	}
	if input.SourceText != sampleInput.SourceText || input.DraftText != sampleInput.DraftText {
		t.Fatal("loadInput() did not return the built-in sample")
	}
}

func TestLoadInput_File_ReturnsParsedPractice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "practice.json")
	content := `{
		"source_text": "El gato duerme.",
		"draft_text": "The cat sleep.",
		"target_rules": [{"verb": "sleep", "tense": "present simple"}]
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	input, err := loadInput(path)
	if err != nil {
		t.Fatalf("loadInput() error = %v", err)
	}
	if input.SourceText != "El gato duerme." || input.DraftText != "The cat sleep." {
		t.Fatalf("loadInput() = %+v, unexpected", input)
	}
	if len(input.TargetRules) != 1 || input.TargetRules[0].Verb != "sleep" {
		t.Fatalf("TargetRules = %v, want one sleep rule", input.TargetRules)
	}
	if input.TargetRules[0].Tense != "present simple" {
		t.Fatalf("Tense = %q, want present simple", input.TargetRules[0].Tense)
	}
}

func TestLoadInput_InvalidJSON_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if _, err := loadInput(path); err == nil {
		t.Fatal("loadInput(invalid) error = nil, want error")
	}
}
