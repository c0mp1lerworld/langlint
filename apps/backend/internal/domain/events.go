package domain

// DomainEvent is a pure, immutable business fact (A4). Concrete events live in
// the root domain package so bounded contexts never import each other (A3).
type DomainEvent interface {
	// EventName returns the stable wire name of the event.
	EventName() string
}

// EventNameIdentityIssued is the wire name of IdentityIssued.
const EventNameIdentityIssued = "identity.issued"

// IdentityIssued is emitted when a portable identity is issued (A9).
type IdentityIssued struct {
	UserID  ID  `json:"user_id"`
	Version int `json:"version"`
}

// EventName returns the stable wire name of the event.
func (IdentityIssued) EventName() string { return EventNameIdentityIssued }
