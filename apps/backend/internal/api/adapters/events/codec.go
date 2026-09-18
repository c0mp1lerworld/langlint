// Package events contains the in-memory event bus and the outbox relay adapters
// for the API entry point (MANIFEST §5.1).
package events

import (
	"encoding/json"
	"fmt"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// MarshalEvent encodes a domain event into its stable wire name and its JSON
// payload, ready to be persisted in the outbox.
func MarshalEvent(event domain.DomainEvent) (string, []byte, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return "", nil, fmt.Errorf("marshal event %s: %w", event.EventName(), err)
	}
	return event.EventName(), payload, nil
}

// UnmarshalEvent rebuilds a domain event from its wire name and JSON payload,
// using the domain registry of concrete events.
func UnmarshalEvent(eventType string, payload []byte) (domain.DomainEvent, error) {
	event, ok := domain.NewEvent(eventType)
	if !ok {
		return nil, fmt.Errorf("unknown event type %q", eventType)
	}
	if err := json.Unmarshal(payload, event); err != nil {
		return nil, fmt.Errorf("unmarshal event %s: %w", eventType, err)
	}
	return event, nil
}
