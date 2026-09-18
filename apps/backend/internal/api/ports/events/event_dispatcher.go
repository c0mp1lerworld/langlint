package events

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// EventDispatcher delivers domain events to their subscribers post-commit
// (§5.1). Services never call it inside a UnitOfWork transaction (AP7).
type EventDispatcher interface {
	Dispatch(ctx context.Context, event domain.DomainEvent) error
	Subscribe(eventName string, handler EventHandler) error
}
