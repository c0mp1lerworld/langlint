package config

import (
	"fmt"
	"os"
)

// Environment variables consumed by the LLM engine. They keep the standard
// `OPENAI_` prefix used by the official SDK so the same .env works for the
// provider client.
const (
	EnvOpenAIAPIKey       = "OPENAI_API_KEY"
	EnvOpenAIModel        = "OPENAI_MODEL"
	EnvOpenAIModelVersion = "OPENAI_MODEL_VERSION"
	EnvOpenAIBaseURL      = "OPENAI_BASE_URL"
)

// EnvPseudonymSecret keys the HMAC-SHA256 that pseudonymizes identifiers before
// they are materialized in analytics (A8).
const EnvPseudonymSecret = "APP_PSEUDONYM_SECRET"

// LoadPseudonymSecret reads the analytics pseudonymization secret. It is
// required and must not be empty, otherwise pseudonyms would be unkeyed.
func LoadPseudonymSecret() (string, error) {
	secret := os.Getenv(EnvPseudonymSecret)
	if secret == "" {
		return "", fmt.Errorf("%s is required", EnvPseudonymSecret)
	}
	return secret, nil
}

// OpenAIConfig is the configuration of the LLM adapter (PRODUCT_DOMAIN §6.1).
type OpenAIConfig struct {
	APIKey       string
	Model        string
	ModelVersion string
	BaseURL      string
}

// LoadOpenAIConfig reads the OpenAI settings from the environment. APIKey and
// Model are required; ModelVersion falls back to Model when unset.
func LoadOpenAIConfig() (OpenAIConfig, error) {
	cfg := OpenAIConfig{
		APIKey:       os.Getenv(EnvOpenAIAPIKey),
		Model:        os.Getenv(EnvOpenAIModel),
		ModelVersion: os.Getenv(EnvOpenAIModelVersion),
		BaseURL:      os.Getenv(EnvOpenAIBaseURL),
	}

	if cfg.APIKey == "" {
		return OpenAIConfig{}, fmt.Errorf("%s is required", EnvOpenAIAPIKey)
	}
	if cfg.Model == "" {
		return OpenAIConfig{}, fmt.Errorf("%s is required", EnvOpenAIModel)
	}
	if cfg.ModelVersion == "" {
		cfg.ModelVersion = cfg.Model
	}

	return cfg, nil
}
