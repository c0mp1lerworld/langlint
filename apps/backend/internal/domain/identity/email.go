package identity

import (
	"net/mail"
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Email is a validated email address. PII: never log it raw (A8).
type Email string

// NewEmail validates raw and returns an Email value object.
func NewEmail(raw string) (Email, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", &domain.ValidationError{Field: "email", Message: "must not be empty"}
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return "", &domain.ValidationError{Field: "email", Message: "must be a valid email address"}
	}

	return Email(trimmed), nil
}

// String returns the raw email address.
func (e Email) String() string {
	return string(e)
}
