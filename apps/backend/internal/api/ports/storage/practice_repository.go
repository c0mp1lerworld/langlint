package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// PracticeRepository persists the Practice aggregate (A2). Implementations live
// in adapters and own the transaction detail via ctxTx (AP8).
type PracticeRepository interface {
	Save(ctx context.Context, practice *practice.Practice) error
	GetByID(ctx context.Context, id domain.ID) (*practice.Practice, error)
	ListByUser(ctx context.Context, userID domain.ID, limit, offset int) ([]practice.Practice, int, error)
	// ListAllByUser returns every non-deleted practice of the user, unpaginated,
	// for the A9 data export.
	ListAllByUser(ctx context.Context, userID domain.ID) ([]practice.Practice, error)
}
