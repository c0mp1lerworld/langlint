package analysis

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewAnalysis_IDGenerationFailure_ReturnsError(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	if _, err := NewAnalysis(domain.MustNewID(), "gpt-4o", "2024-05-13", time.Now().UTC()); err == nil {
		t.Fatal("NewAnalysis() with failing ID generation: want error, got nil")
	}
}
