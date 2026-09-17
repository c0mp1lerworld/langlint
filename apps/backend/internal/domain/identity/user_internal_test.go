package identity

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewUser_IDGenerationFailure_ReturnsError(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	email, err := NewEmail("student@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	if _, err := NewUser(email, time.Now().UTC()); err == nil {
		t.Fatal("NewUser() with failing ID generation: want error, got nil")
	}
}
