package identity

import (
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// DeletionRequest records a user's right-to-be-forgotten request (A9, GDPR
// Art. 17). Deletion is not immediate: the request stays pending until the
// grace period elapses, then the provisioner's execute-deletions job
// materializes it. It carries no tenant_id (AP1).
type DeletionRequest struct {
	ID          domain.ID  `json:"id"`
	UserID      domain.ID  `json:"user_id"`
	RequestedAt time.Time  `json:"requested_at"`
	ExecutedAt  *time.Time `json:"executed_at"`
}

// NewDeletionRequest creates a pending deletion request, generating its UUID v7.
func NewDeletionRequest(userID domain.ID, requestedAt time.Time) (*DeletionRequest, error) {
	if userID.IsZero() {
		return nil, &domain.ValidationError{Field: "user_id", Message: "must not be empty"}
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &DeletionRequest{
		ID:          id,
		UserID:      userID,
		RequestedAt: requestedAt,
	}, nil
}

// Due reports whether the request is past its grace period and still pending.
// An already executed request is never due.
func (d *DeletionRequest) Due(now time.Time, grace time.Duration) bool {
	if d.ExecutedAt != nil {
		return false
	}
	if grace < 0 {
		grace = 0
	}
	return !now.Before(d.RequestedAt.Add(grace))
}

// MarkExecuted records that the request was materialized. Executing twice is an
// invalid state, so a retry cannot delete a user's data a second time.
func (d *DeletionRequest) MarkExecuted(now time.Time) error {
	if d.ExecutedAt != nil {
		return &domain.InvalidStateError{Field: "executed_at", Message: "deletion request already executed"}
	}
	executedAt := now
	d.ExecutedAt = &executedAt
	return nil
}
