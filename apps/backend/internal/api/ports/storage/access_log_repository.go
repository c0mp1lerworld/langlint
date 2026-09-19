package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

// AccessLogRepository is the append-only audit of the user's accesses (A4, A9).
// It intentionally exposes no Update or Delete: history is immutable.
type AccessLogRepository interface {
	Append(ctx context.Context, event *identity.AccessEvent) error
	ListByUser(ctx context.Context, userID domain.ID, limit, offset int) ([]identity.AccessEvent, int, error)
}
