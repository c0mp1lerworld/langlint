package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Environment variables consumed by the provisioner batch jobs. They keep the
// project prefix APP_ (AP-MR4).
const (
	EnvRawRetentionDays  = "APP_RAW_RETENTION_DAYS"
	EnvDeletionGraceDays = "APP_DELETION_GRACE_DAYS"
)

const (
	defaultRawRetentionDays  = 30
	defaultDeletionGraceDays = 30
)

// ProvisionerConfig is the configuration of the batch entry point
// (cmd/provisioner). RawRetention is how long a soft-deleted practice is kept
// before purge-raw-data removes it (A8); DeletionGrace is how long a deletion
// request waits before execute-deletions materializes it (A9).
type ProvisionerConfig struct {
	DatabaseURL     string
	RawRetention    time.Duration
	DeletionGrace   time.Duration
	PseudonymSecret string
}

// LoadProvisionerConfig reads the provisioner settings from the environment.
// DatabaseURL is required; the retention windows default to 30 days.
func LoadProvisionerConfig() (ProvisionerConfig, error) {
	databaseURL, err := LoadDatabaseURL()
	if err != nil {
		return ProvisionerConfig{}, err
	}

	retentionDays, err := loadDays(EnvRawRetentionDays, defaultRawRetentionDays)
	if err != nil {
		return ProvisionerConfig{}, err
	}
	graceDays, err := loadDays(EnvDeletionGraceDays, defaultDeletionGraceDays)
	if err != nil {
		return ProvisionerConfig{}, err
	}

	pseudonymSecret, err := LoadPseudonymSecret()
	if err != nil {
		return ProvisionerConfig{}, err
	}

	return ProvisionerConfig{
		DatabaseURL:     databaseURL,
		RawRetention:    daysToDuration(retentionDays),
		DeletionGrace:   daysToDuration(graceDays),
		PseudonymSecret: pseudonymSecret,
	}, nil
}

// loadDays reads a non-negative number of days from env, falling back to the
// default when unset.
func loadDays(env string, fallback int) (int, error) {
	raw := os.Getenv(env)
	if raw == "" {
		return fallback, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s is not an integer: %w", env, err)
	}
	if days < 0 {
		return 0, fmt.Errorf("%s must not be negative", env)
	}
	return days, nil
}

// daysToDuration converts whole days into a duration.
func daysToDuration(days int) time.Duration {
	return time.Duration(days) * 24 * time.Hour
}
