// Package config loads the backend configuration from the process environment
// (MANIFEST_MONOREPO §3.1: os.Getenv manual, no external config libraries).
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ParseDotEnv parses KEY=VALUE pairs from r. Blank lines and lines whose first
// non-space character is '#' are ignored. An optional `export ` prefix and
// surrounding single or double quotes are supported.
func ParseDotEnv(r io.Reader) (map[string]string, error) {
	values := make(map[string]string)

	scanner := bufio.NewScanner(r)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		raw = strings.TrimPrefix(raw, "export ")
		key, value, found := strings.Cut(raw, "=")
		if !found {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", line)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", line)
		}
		values[key] = unquote(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

// LoadDotEnv reads a .env file and applies its variables to the process
// environment. Variables already present are never overridden, so a real
// deployment takes precedence over the file. A missing file is not an error:
// it means the environment must provide the values.
func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	values, err := ParseDotEnv(file)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return nil
}

// unquote removes a matching pair of surrounding single or double quotes.
func unquote(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
