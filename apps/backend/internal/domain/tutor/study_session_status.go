package tutor

// StudySessionStatus is the lifecycle state of a StudySession (PRODUCT_DOMAIN
// §12.1). Generated sessions start in generated, become active once the learner
// starts them and finish completed.
type StudySessionStatus string

// Lifecycle states of a StudySession. The values are the wire representation.
const (
	StudySessionStatusGenerated StudySessionStatus = "generated"
	StudySessionStatusActive    StudySessionStatus = "active"
	StudySessionStatusCompleted StudySessionStatus = "completed"
)

// IsValid reports whether the status is one of the known lifecycle values.
func (s StudySessionStatus) IsValid() bool {
	switch s {
	case StudySessionStatusGenerated, StudySessionStatusActive, StudySessionStatusCompleted:
		return true
	default:
		return false
	}
}

// String returns the wire representation of the status.
func (s StudySessionStatus) String() string {
	return string(s)
}
