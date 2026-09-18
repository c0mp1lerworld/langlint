package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotEnv_ParsesPairsAndIgnoresNoise(t *testing.T) {
	input := strings.Join([]string{
		"# comment",
		"",
		"OPENAI_MODEL=gpt-4o-mini",
		"export OPENAI_MODEL_VERSION=\"gpt-4o-mini-2024-07-18\"",
		"OPENAI_BASE_URL='https://example.test/v1'",
		"OPENAI_API_KEY=sk-secret-value",
	}, "\n")

	values, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDotEnv() error = %v", err)
	}

	want := map[string]string{
		"OPENAI_MODEL":         "gpt-4o-mini",
		"OPENAI_MODEL_VERSION": "gpt-4o-mini-2024-07-18",
		"OPENAI_BASE_URL":      "https://example.test/v1",
		"OPENAI_API_KEY":       "sk-secret-value",
	}
	for key, value := range want {
		if got := values[key]; got != value {
			t.Fatalf("values[%q] = %q, want %q", key, got, value)
		}
	}
	if len(values) != len(want) {
		t.Fatalf("len(values) = %d, want %d", len(values), len(want))
	}
}

func TestParseDotEnv_InvalidLine_ReturnsError(t *testing.T) {
	if _, err := ParseDotEnv(strings.NewReader("not-a-pair\n")); err == nil {
		t.Fatal("ParseDotEnv(invalid) error = nil, want error")
	}
}

func TestParseDotEnv_EmptyKey_ReturnsError(t *testing.T) {
	if _, err := ParseDotEnv(strings.NewReader("=value\n")); err == nil {
		t.Fatal("ParseDotEnv(empty key) error = nil, want error")
	}
}

func TestLoadDotEnv_MissingFile_NoError(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "absent.env")); err != nil {
		t.Fatalf("LoadDotEnv(missing) error = %v", err)
	}
}

func TestLoadDotEnv_AppliesValuesAndKeepsExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "LLMCHECK_NEW=demo\nLLMCHECK_EXISTING=from-file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	os.Unsetenv("LLMCHECK_NEW")
	t.Cleanup(func() { os.Unsetenv("LLMCHECK_NEW") })
	t.Setenv("LLMCHECK_EXISTING", "from-env")

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() error = %v", err)
	}
	if got := os.Getenv("LLMCHECK_NEW"); got != "demo" {
		t.Fatalf("LLMCHECK_NEW = %q, want demo", got)
	}
	if got := os.Getenv("LLMCHECK_EXISTING"); got != "from-env" {
		t.Fatalf("LLMCHECK_EXISTING = %q, want from-env (env must win)", got)
	}
}

func TestLoadOpenAIConfig_MissingRequired_ReturnsError(t *testing.T) {
	t.Setenv(EnvOpenAIAPIKey, "")
	t.Setenv(EnvOpenAIModel, "gpt-4o-mini")

	if _, err := LoadOpenAIConfig(); err == nil {
		t.Fatal("LoadOpenAIConfig() error = nil, want error")
	}
}

func TestLoadOpenAIConfig_VersionFallsBackToModel(t *testing.T) {
	t.Setenv(EnvOpenAIAPIKey, "sk-test")
	t.Setenv(EnvOpenAIModel, "gpt-4o-mini")
	t.Setenv(EnvOpenAIModelVersion, "")

	cfg, err := LoadOpenAIConfig()
	if err != nil {
		t.Fatalf("LoadOpenAIConfig() error = %v", err)
	}
	if cfg.ModelVersion != "gpt-4o-mini" {
		t.Fatalf("ModelVersion = %q, want gpt-4o-mini", cfg.ModelVersion)
	}
}

func TestLoadOpenAIConfig_ReadsAllValues(t *testing.T) {
	t.Setenv(EnvOpenAIAPIKey, "sk-test")
	t.Setenv(EnvOpenAIModel, "gpt-4o-mini")
	t.Setenv(EnvOpenAIModelVersion, "gpt-4o-mini-2024-07-18")
	t.Setenv(EnvOpenAIBaseURL, "https://example.test/v1")

	cfg, err := LoadOpenAIConfig()
	if err != nil {
		t.Fatalf("LoadOpenAIConfig() error = %v", err)
	}
	if cfg.APIKey != "sk-test" || cfg.Model != "gpt-4o-mini" || cfg.ModelVersion != "gpt-4o-mini-2024-07-18" || cfg.BaseURL != "https://example.test/v1" {
		t.Fatalf("LoadOpenAIConfig() = %+v, unexpected", cfg)
	}
}
