package analysis

import (
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Analysis is the aggregate root of the AI-generated breakdown of a practice
// (PRODUCT_DOMAIN §4.2.3). It references the practice by ID only, so it never
// imports the practice bounded context (A3).
type Analysis struct {
	ID           domain.ID      `json:"id"`
	PracticeID   domain.ID      `json:"practice_id"`
	Fragments    []Fragment     `json:"fragments"`
	Model        string         `json:"model"`
	ModelVersion string         `json:"model_version"`
	Status       AnalysisStatus `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
}

// generateID is a seam for deterministic tests; it defaults to domain.NewID.
var generateID = domain.NewID

// NewAnalysis creates an Analysis in pending status, generating its UUID v7.
func NewAnalysis(practiceID domain.ID, model, modelVersion string, createdAt time.Time) (*Analysis, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &Analysis{
		ID:           id,
		PracticeID:   practiceID,
		Model:        model,
		ModelVersion: modelVersion,
		Status:       AnalysisStatusPending,
		CreatedAt:    createdAt,
	}, nil
}

// Complete transitions pending -> completed. Fragments must not be empty when
// an analysis is completed (PRODUCT_DOMAIN §4.2.3).
func (a *Analysis) Complete(fragments []Fragment) error {
	if a.Status != AnalysisStatusPending {
		return &domain.InvalidStateError{Field: "status", Message: "must be pending to be completed"}
	}
	if len(fragments) == 0 {
		return &domain.ValidationError{Field: "fragments", Message: "must not be empty when completed"}
	}
	a.Fragments = fragments
	a.Status = AnalysisStatusCompleted
	return nil
}

// Fail transitions pending -> failed.
func (a *Analysis) Fail() error {
	if a.Status != AnalysisStatusPending {
		return &domain.InvalidStateError{Field: "status", Message: "must be pending to fail"}
	}
	a.Status = AnalysisStatusFailed
	return nil
}
