package practice

// PracticeStatus is the lifecycle state of a Practice (PRODUCT_DOMAIN §4.2.2).
type PracticeStatus string

// Lifecycle states of a Practice. The values are the wire representation.
const (
	PracticeStatusDraft     PracticeStatus = "draft"
	PracticeStatusAnalyzing PracticeStatus = "analyzing"
	PracticeStatusCompleted PracticeStatus = "completed"
	PracticeStatusFailed    PracticeStatus = "failed"
)

// IsValid reports whether the status is one of the known lifecycle values.
func (s PracticeStatus) IsValid() bool {
	switch s {
	case PracticeStatusDraft, PracticeStatusAnalyzing, PracticeStatusCompleted, PracticeStatusFailed:
		return true
	default:
		return false
	}
}

// String returns the wire representation of the status.
func (s PracticeStatus) String() string {
	return string(s)
}
