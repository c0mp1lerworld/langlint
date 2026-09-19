package config_test

import (
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
)

func TestLoadServerConfig_AllValues_ReturnsConfig(t *testing.T) {
	userID := domain.MustNewID()
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvHTTPAddr, ":9999")
	t.Setenv(config.EnvUserID, userID.String())
	t.Setenv(config.EnvLLMTimeout, "5s")
	t.Setenv(config.EnvCORSAllowedOrigins, "http://localhost:3000, https://app.langlint.dev")

	cfg, err := config.LoadServerConfig()
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://localhost/langlint" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.HTTPAddr != ":9999" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.UserID != userID {
		t.Fatalf("UserID = %v, want %v", cfg.UserID, userID)
	}
	if cfg.LLMTimeout != 5*time.Second {
		t.Fatalf("LLMTimeout = %v, want 5s", cfg.LLMTimeout)
	}
	if len(cfg.CORSAllowedOrigins) != 2 ||
		cfg.CORSAllowedOrigins[0] != "http://localhost:3000" ||
		cfg.CORSAllowedOrigins[1] != "https://app.langlint.dev" {
		t.Fatalf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
}

func TestLoadServerConfig_Defaults_AppliesHTTPAddrAndTimeout(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvUserID, domain.MustNewID().String())

	cfg, err := config.LoadServerConfig()
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.LLMTimeout != 60*time.Second {
		t.Fatalf("LLMTimeout = %v, want 60s", cfg.LLMTimeout)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("CORSAllowedOrigins = %v, want [http://localhost:3000]", cfg.CORSAllowedOrigins)
	}
}

func TestLoadServerConfig_MissingDatabaseURL_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "")
	t.Setenv(config.EnvUserID, domain.MustNewID().String())

	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatal("LoadServerConfig() error = nil, want error")
	}
}

func TestLoadServerConfig_MissingUserID_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvUserID, "")

	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatal("LoadServerConfig() error = nil, want error")
	}
}

func TestLoadServerConfig_InvalidUserID_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvUserID, "not-a-uuid")

	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatal("LoadServerConfig() error = nil, want error")
	}
}

func TestLoadServerConfig_InvalidTimeout_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvUserID, domain.MustNewID().String())
	t.Setenv(config.EnvLLMTimeout, "not-a-duration")

	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatal("LoadServerConfig() error = nil, want error")
	}
}

func TestLoadDatabaseURL_Missing_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "")

	if _, err := config.LoadDatabaseURL(); err == nil {
		t.Fatal("LoadDatabaseURL() error = nil, want error")
	}
}
