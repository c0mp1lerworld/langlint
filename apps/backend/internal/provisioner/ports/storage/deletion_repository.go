package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

// DeletionRepository drives the right-to-be-forgotten flow (A9).
type DeletionRepository interface {
	// ListPending returns the deletion requests that have not been executed.
	ListPending(ctx context.Context) ([]identity.DeletionRequest, error)
	// Execute purges every datum owned by the request's user and marks the
	// request executed in a single transaction.
	Execute(ctx context.Context, req *identity.DeletionRequest) error
}
