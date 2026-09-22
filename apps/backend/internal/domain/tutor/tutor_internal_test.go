package tutor

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewStudySession_IDGenerationFailure_ReturnsError(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	entry, err := NewWeaknessEntry(domain.ErrorPatternCodeWordOrder, domain.ErrorPatternSeverityModerate, 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewWeaknessEntry() error = %v", err)
	}
	profile, err := NewWeaknessProfile([]WeaknessEntry{entry})
	if err != nil {
		t.Fatalf("NewWeaknessProfile() error = %v", err)
	}
	trap, err := NewTrap(domain.ErrorPatternCodeWordOrder, "a trap")
	if err != nil {
		t.Fatalf("NewTrap() error = %v", err)
	}
	exercise, err := NewExercise(ExerciseKindOpen, "prompt", "answer")
	if err != nil {
		t.Fatalf("NewExercise() error = %v", err)
	}

	if _, err := NewStudySession(domain.MustNewID(), profile, "theory", []Trap{trap}, []Exercise{exercise}, time.Now().UTC()); err == nil {
		t.Fatal("NewStudySession() with failing ID generation: want error, got nil")
	}
}
