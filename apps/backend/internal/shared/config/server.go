package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Environment variables consumed by the HTTP API entry point. They keep the
// project prefix APP_ (AP-MR4).
const (
	EnvDatabaseURL        = "APP_DATABASE_URL"
	EnvHTTPAddr           = "APP_HTTP_ADDR"
	EnvUserID             = "APP_USER_ID"
	EnvLLMTimeout         = "APP_LLM_TIMEOUT"
	EnvCORSAllowedOrigins = "APP_CORS_ALLOWED_ORIGINS"
)

const (
	defaultHTTPAddr   = ":8080"
	defaultLLMTimeout = 60 * time.Second
	defaultCORSOrigin = "http://localhost:3000"
)

// ServerConfig is the configuration of the HTTP API entry point (cmd/api).
// UserID resolves the single-user identity of the MVP: the wire contract has no
// authentication (PRODUCT_DOMAIN §3.2), so the current user comes from config.
// CORSAllowedOrigins lists the browser origins allowed to call the API (F3).
type ServerConfig struct {
	DatabaseURL        string
	HTTPAddr           string
	UserID             domain.ID
	LLMTimeout         time.Duration
	CORSAllowedOrigins []string
}

// LoadDatabaseURL reads the Postgres connection string. It is required.
func LoadDatabaseURL() (string, error) {
	url := os.Getenv(EnvDatabaseURL)
	if url == "" {
		return "", fmt.Errorf("%s is required", EnvDatabaseURL)
	}
	return url, nil
}

// LoadServerConfig reads the API server settings from the environment.
// DatabaseURL and UserID are required; HTTPAddr defaults to :8080 and
// LLMTimeout to 60s.
func LoadServerConfig() (ServerConfig, error) {
	databaseURL, err := LoadDatabaseURL()
	if err != nil {
		return ServerConfig{}, err
	}

	rawUserID := os.Getenv(EnvUserID)
	if rawUserID == "" {
		return ServerConfig{}, fmt.Errorf("%s is required", EnvUserID)
	}
	userID, err := domain.ParseID(rawUserID)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("%s is not a valid UUID: %w", EnvUserID, err)
	}

	cfg := ServerConfig{
		DatabaseURL: databaseURL,
		HTTPAddr:    os.Getenv(EnvHTTPAddr),
		UserID:      userID,
		LLMTimeout:  defaultLLMTimeout,
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = defaultHTTPAddr
	}

	if rawTimeout := os.Getenv(EnvLLMTimeout); rawTimeout != "" {
		timeout, err := time.ParseDuration(rawTimeout)
		if err != nil {
			return ServerConfig{}, fmt.Errorf("%s is not a valid duration: %w", EnvLLMTimeout, err)
		}
		cfg.LLMTimeout = timeout
	}

	cfg.CORSAllowedOrigins = parseOrigins(os.Getenv(EnvCORSAllowedOrigins))

	return cfg, nil
}

// parseOrigins splits a comma-separated allow-list, trimming blanks. When raw is
// empty the local frontend origin is used as the development default.
func parseOrigins(raw string) []string {
	if raw == "" {
		return []string{defaultCORSOrigin}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
