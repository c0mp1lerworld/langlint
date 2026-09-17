package practice

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewPractice_IDGenerationFailure_ReturnsError(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	source, err := NewSourceText("El gato duerme.")
	if err != nil {
		t.Fatalf("NewSourceText() error = %v", err)
	}
	draft, err := NewDraftText("The cat sleep.")
	if err != nil {
		t.Fatalf("NewDraftText() error = %v", err)
	}
	rule, err := NewTargetRule("sleep", "present", "")
	if err != nil {
		t.Fatalf("NewTargetRule() error = %v", err)
	}

	if _, err := NewPractice(domain.MustNewID(), source, draft, []TargetRule{rule}, time.Now().UTC()); err == nil {
		t.Fatal("NewPractice() with failing ID generation: want error, got nil")
	}
}
