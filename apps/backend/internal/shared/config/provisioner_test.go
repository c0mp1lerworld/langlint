package config_test

import (
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
)

func TestLoadProvisionerConfig_AllValues_ReturnsConfig(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvRawRetentionDays, "7")
	t.Setenv(config.EnvDeletionGraceDays, "45")

	cfg, err := config.LoadProvisionerConfig()
	if err != nil {
		t.Fatalf("LoadProvisionerConfig() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://localhost/langlint" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.RawRetention != 7*24*time.Hour {
		t.Fatalf("RawRetention = %v, want 168h", cfg.RawRetention)
	}
	if cfg.DeletionGrace != 45*24*time.Hour {
		t.Fatalf("DeletionGrace = %v, want 1080h", cfg.DeletionGrace)
	}
}

func TestLoadProvisionerConfig_Defaults_AppliesThirtyDays(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvRawRetentionDays, "")
	t.Setenv(config.EnvDeletionGraceDays, "")

	cfg, err := config.LoadProvisionerConfig()
	if err != nil {
		t.Fatalf("LoadProvisionerConfig() error = %v", err)
	}

	if cfg.RawRetention != 30*24*time.Hour {
		t.Fatalf("RawRetention = %v, want 720h", cfg.RawRetention)
	}
	if cfg.DeletionGrace != 30*24*time.Hour {
		t.Fatalf("DeletionGrace = %v, want 720h", cfg.DeletionGrace)
	}
}

func TestLoadProvisionerConfig_ZeroDays_Allowed(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvRawRetentionDays, "0")

	cfg, err := config.LoadProvisionerConfig()
	if err != nil {
		t.Fatalf("LoadProvisionerConfig() error = %v", err)
	}
	if cfg.RawRetention != 0 {
		t.Fatalf("RawRetention = %v, want 0", cfg.RawRetention)
	}
}

func TestLoadProvisionerConfig_MissingDatabaseURL_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "")

	if _, err := config.LoadProvisionerConfig(); err == nil {
		t.Fatal("LoadProvisionerConfig() error = nil, want error")
	}
}

func TestLoadProvisionerConfig_InvalidDays_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvRawRetentionDays, "soon")

	if _, err := config.LoadProvisionerConfig(); err == nil {
		t.Fatal("LoadProvisionerConfig() error = nil, want error")
	}
}

func TestLoadProvisionerConfig_NegativeDays_ReturnsError(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://localhost/langlint")
	t.Setenv(config.EnvDeletionGraceDays, "-1")

	if _, err := config.LoadProvisionerConfig(); err == nil {
		t.Fatal("LoadProvisionerConfig() error = nil, want error")
	}
}
