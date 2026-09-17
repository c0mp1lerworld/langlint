package practice

import (
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// DraftText is the user's experimental translation into English (PRODUCT_DOMAIN §4.3).
type DraftText string

// NewDraftText validates raw and returns a DraftText value object.
func NewDraftText(raw string) (DraftText, error) {
	if strings.TrimSpace(raw) == "" {
		return "", &domain.ValidationError{Field: "draft_text", Message: "must not be empty"}
	}
	return DraftText(raw), nil
}

// String returns the raw draft text.
func (d DraftText) String() string {
	return string(d)
}
