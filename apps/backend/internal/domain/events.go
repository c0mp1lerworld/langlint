package domain

import "time"

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

// EventNamePracticeCreated is the wire name of PracticeCreated.
const EventNamePracticeCreated = "practice.created"

// PracticeCreated is emitted when a practice is created (triggers analysis).
type PracticeCreated struct {
	PracticeID ID  `json:"practice_id"`
	UserID     ID  `json:"user_id"`
	Version    int `json:"version"`
}

// EventName returns the stable wire name of the event.
func (PracticeCreated) EventName() string { return EventNamePracticeCreated }

// EventNameAnalysisRequested is the wire name of AnalysisRequested.
const EventNameAnalysisRequested = "analysis.requested"

// AnalysisRequested is emitted when the user asks to analyze a practice
// (triggers the asynchronous analysis). It is distinct from PracticeCreated:
// creating a practice leaves it in draft and does not start the LLM.
type AnalysisRequested struct {
	PracticeID ID  `json:"practice_id"`
	UserID     ID  `json:"user_id"`
	Version    int `json:"version"`
}

// EventName returns the stable wire name of the event.
func (AnalysisRequested) EventName() string { return EventNameAnalysisRequested }

// EventNameAnalysisCompleted is the wire name of AnalysisCompleted.
const EventNameAnalysisCompleted = "analysis.completed"

// AnalysisCompleted is emitted when an analysis finishes successfully.
// It feeds the analytics bounded context (A3).
type AnalysisCompleted struct {
	AnalysisID    ID             `json:"analysis_id"`
	PracticeID    ID             `json:"practice_id"`
	UserID        ID             `json:"user_id"`
	ErrorPatterns []ErrorPattern `json:"error_patterns"`
	Version       int            `json:"version"`
}

// EventName returns the stable wire name of the event.
func (AnalysisCompleted) EventName() string { return EventNameAnalysisCompleted }

// EventNameWeaknessDetected is the wire name of WeaknessDetected.
const EventNameWeaknessDetected = "analytics.weakness_detected"

// WeaknessDetected is emitted by the analytics bounded context when a learner's
// error frequency for one or more patterns reaches the weakness threshold
// (PRODUCT_DOMAIN §12.1, checklist 8.2). It feeds the tutor bounded context.
type WeaknessDetected struct {
	UserID        ID             `json:"user_id"`
	ErrorPatterns []ErrorPattern `json:"error_patterns"`
	Window        Window         `json:"window"`
	Version       int            `json:"version"`
}

// EventName returns the stable wire name of the event.
func (WeaknessDetected) EventName() string { return EventNameWeaknessDetected }

// EventNameAnalysisFailed is the wire name of AnalysisFailed.
const EventNameAnalysisFailed = "analysis.failed"

// AnalysisFailed is emitted when an analysis cannot be completed.
type AnalysisFailed struct {
	AnalysisID ID     `json:"analysis_id"`
	PracticeID ID     `json:"practice_id"`
	Reason     string `json:"reason"`
	Version    int    `json:"version"`
}

// EventName returns the stable wire name of the event.
func (AnalysisFailed) EventName() string { return EventNameAnalysisFailed }

// OutboxEvent is a persisted domain event awaiting publication (§5.1). Payload
// holds the JSON encoding of the concrete event; the relay marks PublishedAt
// once dispatched and counts failed Attempts.
type OutboxEvent struct {
	ID          ID         `json:"id"`
	EventType   string     `json:"event_type"`
	Payload     []byte     `json:"payload"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
	Attempts    int        `json:"attempts"`
}

// NewEvent returns an empty event of the given wire name, or false when the
// name is unknown. It is the single registry of concrete domain events and lets
// adapters rebuild an event from the outbox payload (§5.1).
func NewEvent(eventType string) (DomainEvent, bool) {
	switch eventType {
	case EventNameIdentityIssued:
		return &IdentityIssued{}, true
	case EventNamePracticeCreated:
		return &PracticeCreated{}, true
	case EventNameAnalysisRequested:
		return &AnalysisRequested{}, true
	case EventNameAnalysisCompleted:
		return &AnalysisCompleted{}, true
	case EventNameAnalysisFailed:
		return &AnalysisFailed{}, true
	case EventNameWeaknessDetected:
		return &WeaknessDetected{}, true
	default:
		return nil, false
	}
}
