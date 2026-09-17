package analytics

import (
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// ErrorMetric is the materialized frequency of an error pattern for a user in a
// time window (PRODUCT_DOMAIN §4.2.4). It is keyed by (UserID, Code, Window) and
// upserted by the AnalysisCompleted handler.
type ErrorMetric struct {
	UserID     domain.ID               `json:"user_id"`
	Code       domain.ErrorPatternCode `json:"code"`
	Window     Window                  `json:"window"`
	Count      int                     `json:"count"`
	LastSeenAt time.Time               `json:"last_seen_at"`
}

// NewErrorMetric creates a metric for the first occurrence of a pattern.
func NewErrorMetric(userID domain.ID, code domain.ErrorPatternCode, window Window, lastSeenAt time.Time) (*ErrorMetric, error) {
	if !code.IsValid() {
		return nil, &domain.ValidationError{Field: "code", Message: "must be a known error pattern code"}
	}
	if !window.IsValid() {
		return nil, &domain.ValidationError{Field: "window", Message: "must be a known window"}
	}
	return &ErrorMetric{
		UserID:     userID,
		Code:       code,
		Window:     window,
		Count:      1,
		LastSeenAt: lastSeenAt,
	}, nil
}

// Record registers a new occurrence of the pattern.
func (m *ErrorMetric) Record(now time.Time) {
	m.Count++
	m.LastSeenAt = now
}
