package identity

import (
	"strings"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// AccessEvent is an append-only record of a user's access to their own data
// (A9, A4). It is never updated nor deleted: the audit is immutable.
type AccessEvent struct {
	ID           domain.ID `json:"id"`
	UserID       domain.ID `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	OccurredAt   time.Time `json:"occurred_at"`
}

// NewAccessEvent records an access, generating its UUID v7. userID must be set
// and action non-empty; resource_type and resource_id are free-form (an empty
// value means "not applicable").
func NewAccessEvent(userID domain.ID, action, resourceType, resourceID string, occurredAt time.Time) (*AccessEvent, error) {
	if userID.IsZero() {
		return nil, &domain.ValidationError{Field: "user_id", Message: "must not be empty"}
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return nil, &domain.ValidationError{Field: "action", Message: "must not be empty"}
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &AccessEvent{
		ID:           id,
		UserID:       userID,
		Action:       action,
		ResourceType: strings.TrimSpace(resourceType),
		ResourceID:   strings.TrimSpace(resourceID),
		OccurredAt:   occurredAt,
	}, nil
}
