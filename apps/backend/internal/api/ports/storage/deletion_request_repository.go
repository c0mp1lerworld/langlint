package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

// DeletionRequestRepository records the A9 right-to-be-forgotten requests. The
// provisioner's execute-deletions job materializes them after the grace period.
type DeletionRequestRepository interface {
	// Append stores a new pending deletion request.
	Append(ctx context.Context, req *identity.DeletionRequest) error
	// HasPending reports whether the user already has an unexecuted request,
	// so a repeated DELETE /me/data stays idempotent (AP4).
	HasPending(ctx context.Context, userID domain.ID) (bool, error)
}
