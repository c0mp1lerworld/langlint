package practice

import (
	"strings"
	"unicode/utf8"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// SourceText is the base text in Spanish to be translated (PRODUCT_DOMAIN §4.3).
type SourceText string

// NewSourceText validates raw and returns a SourceText value object.
func NewSourceText(raw string) (SourceText, error) {
	if strings.TrimSpace(raw) == "" {
		return "", &domain.ValidationError{Field: "source_text", Message: "must not be empty"}
	}
	if utf8.RuneCountInString(raw) > maxTextRunes {
		return "", &domain.ValidationError{Field: "source_text", Message: "must not exceed 2000 characters"}
	}
	return SourceText(raw), nil
}

// String returns the raw source text.
func (s SourceText) String() string {
	return string(s)
}
