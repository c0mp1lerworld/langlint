package identity

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewDeletionRequest_IDGenerationFailure_ReturnsError(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	if _, err := NewDeletionRequest(domain.MustNewID(), time.Now().UTC()); err == nil {
		t.Fatal("NewDeletionRequest() with failing ID generation: want error, got nil")
	}
}
