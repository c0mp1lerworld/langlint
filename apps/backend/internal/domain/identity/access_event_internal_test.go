package identity

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewAccessEvent_IDGenerationFailure_ReturnsError(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	if _, err := NewAccessEvent(domain.MustNewID(), "GET", "", "", time.Now().UTC()); err == nil {
		t.Fatal("NewAccessEvent() with failing ID generation: want error, got nil")
	}
}
