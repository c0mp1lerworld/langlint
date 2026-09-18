package events

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// EventHandler consumes a single domain event. Handlers must be idempotent:
// running them twice has no adverse effect (§5.1).
type EventHandler interface {
	Handle(ctx context.Context, event domain.DomainEvent) error
}
