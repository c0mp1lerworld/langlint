package analysis

// AnalysisStatus is the lifecycle state of an Analysis (PRODUCT_DOMAIN §4.2.3).
type AnalysisStatus string

// Lifecycle states of an Analysis. The values are the wire representation.
const (
	AnalysisStatusPending   AnalysisStatus = "pending"
	AnalysisStatusCompleted AnalysisStatus = "completed"
	AnalysisStatusFailed    AnalysisStatus = "failed"
)

// IsValid reports whether the status is one of the known lifecycle values.
func (s AnalysisStatus) IsValid() bool {
	switch s {
	case AnalysisStatusPending, AnalysisStatusCompleted, AnalysisStatusFailed:
		return true
	default:
		return false
	}
}

// String returns the wire representation of the status.
func (s AnalysisStatus) String() string {
	return string(s)
}
