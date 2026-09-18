package events

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Outbox appends a domain event within the caller's transaction so data and
// event commit atomically (AP7, §5.1).
type Outbox interface {
	Append(ctx context.Context, event domain.DomainEvent) error
}
