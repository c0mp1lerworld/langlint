package tutor

import (
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Trap is a common mistake the learner keeps making, anchored to an error
// pattern (PRODUCT_DOMAIN §12.1, checklist 8.1.2: "trampas comunes").
type Trap struct {
	Code        domain.ErrorPatternCode `json:"code"`
	Description string                  `json:"description"`
}

// NewTrap validates the code and description and returns a Trap value object.
func NewTrap(code domain.ErrorPatternCode, description string) (Trap, error) {
	if !code.IsValid() {
		return Trap{}, &domain.ValidationError{Field: "trap.code", Message: "must be a known error pattern code"}
	}
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return Trap{}, &domain.ValidationError{Field: "trap.description", Message: "must not be empty"}
	}
	return Trap{Code: code, Description: trimmed}, nil
}
