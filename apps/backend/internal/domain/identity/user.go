package identity

import (
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// User is the portable, cross-tenant identity that owns the study data (A9).
// It intentionally carries no tenant_id (AP1).
type User struct {
	ID        domain.ID `json:"id"`
	Email     Email     `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// generateID is a seam for deterministic tests; it defaults to domain.NewID.
var generateID = domain.NewID

// NewUser creates a portable identity, generating its UUID v7 identifier.
func NewUser(email Email, createdAt time.Time) (*User, error) {
	if email == "" {
		return nil, &domain.ValidationError{Field: "email", Message: "must not be empty"}
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &User{ID: id, Email: email, CreatedAt: createdAt}, nil
}
