package analytics

import (
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// ProgressMetric is the temporal progress series for a user in a time window
// (PRODUCT_DOMAIN §4.2.4). Accuracy is derived from the raw counts.
type ProgressMetric struct {
	UserID         domain.ID `json:"user_id"`
	Window         Window    `json:"window"`
	TotalFragments int       `json:"total_fragments"`
	ErrorCount     int       `json:"error_count"`
	Accuracy       float64   `json:"accuracy"`
}

// NewProgressMetric creates a progress metric, deriving Accuracy from the counts.
func NewProgressMetric(userID domain.ID, window Window, totalFragments, errorCount int) (*ProgressMetric, error) {
	if !window.IsValid() {
		return nil, &domain.ValidationError{Field: "window", Message: "must be a known window"}
	}
	if totalFragments < 0 {
		return nil, &domain.ValidationError{Field: "total_fragments", Message: "must not be negative"}
	}
	if errorCount < 0 {
		return nil, &domain.ValidationError{Field: "error_count", Message: "must not be negative"}
	}
	return &ProgressMetric{
		UserID:         userID,
		Window:         window,
		TotalFragments: totalFragments,
		ErrorCount:     errorCount,
		Accuracy:       accuracy(totalFragments, errorCount),
	}, nil
}

// accuracy returns 1 - errorCount/totalFragments clamped to [0, 1]. With no
// fragments there is nothing to be wrong about, so accuracy is 1.
func accuracy(totalFragments, errorCount int) float64 {
	if totalFragments == 0 {
		return 1
	}
	value := 1 - float64(errorCount)/float64(totalFragments)
	if value < 0 {
		return 0
	}
	return value
}
