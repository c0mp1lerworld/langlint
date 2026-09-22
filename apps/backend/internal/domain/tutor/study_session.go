package tutor

import (
	"strings"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// StudySession is the aggregate root of the adaptive tutor (PRODUCT_DOMAIN §12.1,
// checklist 8.1.2). It bundles the summarized theory, the learner's common traps
// and interactive exercises generated from their weakness profile.
type StudySession struct {
	ID        domain.ID          `json:"id"`
	UserID    domain.ID          `json:"user_id"`
	Profile   WeaknessProfile    `json:"profile"`
	Theory    string             `json:"theory"`
	Traps     []Trap             `json:"traps"`
	Exercises []Exercise         `json:"exercises"`
	Status    StudySessionStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
}

// generateID is a seam for deterministic tests; it defaults to domain.NewID.
var generateID = domain.NewID

// NewStudySession creates a session in generated status, generating its UUID v7.
// The profile, theory, traps and exercises must not be empty.
func NewStudySession(userID domain.ID, profile WeaknessProfile, theory string, traps []Trap, exercises []Exercise, createdAt time.Time) (*StudySession, error) {
	if len(profile.Entries) == 0 {
		return nil, &domain.ValidationError{Field: "profile", Message: "must not be empty"}
	}
	if strings.TrimSpace(theory) == "" {
		return nil, &domain.ValidationError{Field: "theory", Message: "must not be empty"}
	}
	if len(traps) == 0 {
		return nil, &domain.ValidationError{Field: "traps", Message: "must not be empty"}
	}
	if len(exercises) == 0 {
		return nil, &domain.ValidationError{Field: "exercises", Message: "must not be empty"}
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &StudySession{
		ID:        id,
		UserID:    userID,
		Profile:   profile,
		Theory:    strings.TrimSpace(theory),
		Traps:     traps,
		Exercises: exercises,
		Status:    StudySessionStatusGenerated,
		CreatedAt: createdAt,
	}, nil
}

// Start transitions generated -> active.
func (s *StudySession) Start() error {
	if s.Status != StudySessionStatusGenerated {
		return &domain.InvalidStateError{Field: "status", Message: "must be generated to start"}
	}
	s.Status = StudySessionStatusActive
	return nil
}

// Complete transitions active -> completed.
func (s *StudySession) Complete() error {
	if s.Status != StudySessionStatusActive {
		return &domain.InvalidStateError{Field: "status", Message: "must be active to be completed"}
	}
	s.Status = StudySessionStatusCompleted
	return nil
}
