package identity_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

const testGrace = 30 * 24 * time.Hour

func mustDeletionRequest(t *testing.T, userID domain.ID, requestedAt time.Time) *identity.DeletionRequest {
	t.Helper()
	req, err := identity.NewDeletionRequest(userID, requestedAt)
	if err != nil {
		t.Fatalf("NewDeletionRequest() error = %v", err)
	}
	return req
}

func TestDeletionRequest_NewDeletionRequest_SetsUserAndRequestedAt(t *testing.T) {
	userID := domain.MustNewID()
	requestedAt := time.Date(2026, time.September, 19, 10, 0, 0, 0, time.UTC)

	req := mustDeletionRequest(t, userID, requestedAt)

	if req.UserID != userID {
		t.Fatalf("UserID = %s, want %s", req.UserID, userID)
	}
	if !req.RequestedAt.Equal(requestedAt) {
		t.Fatalf("RequestedAt = %v, want %v", req.RequestedAt, requestedAt)
	}
	if !req.ID.IsValid() {
		t.Fatalf("ID = %s, want a valid UUID v7", req.ID)
	}
	if req.ExecutedAt != nil {
		t.Fatalf("ExecutedAt = %v, want nil for a fresh request", req.ExecutedAt)
	}
}

func TestDeletionRequest_NewDeletionRequest_ZeroUserID_ReturnsValidationError(t *testing.T) {
	_, err := identity.NewDeletionRequest(domain.ID{}, time.Now().UTC())
	if err == nil {
		t.Fatal("NewDeletionRequest() with zero user id: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewDeletionRequest() error = %T, want *domain.ValidationError", err)
	}
}

func TestDeletionRequest_Due_WithinGrace_ReturnsFalse(t *testing.T) {
	requestedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)

	if req.Due(requestedAt.Add(29*24*time.Hour), testGrace) {
		t.Fatal("Due() within grace: want false, got true")
	}
}

func TestDeletionRequest_Due_PastGrace_ReturnsTrue(t *testing.T) {
	requestedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)

	if !req.Due(requestedAt.Add(31*24*time.Hour), testGrace) {
		t.Fatal("Due() past grace: want true, got false")
	}
}

func TestDeletionRequest_Due_ExactlyAtGraceBoundary_ReturnsTrue(t *testing.T) {
	requestedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)

	if !req.Due(requestedAt.Add(testGrace), testGrace) {
		t.Fatal("Due() exactly at grace: want true, got false")
	}
}

func TestDeletionRequest_Due_NegativeGrace_TreatedAsZero(t *testing.T) {
	requestedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)

	if !req.Due(requestedAt, -time.Hour) {
		t.Fatal("Due() with negative grace: want true, got false")
	}
	if req.Due(requestedAt.Add(-time.Second), -time.Hour) {
		t.Fatal("Due() before requested_at with negative grace: want false, got true")
	}
}

func TestDeletionRequest_Due_AlreadyExecuted_ReturnsFalse(t *testing.T) {
	requestedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)
	if err := req.MarkExecuted(requestedAt.Add(31 * 24 * time.Hour)); err != nil {
		t.Fatalf("MarkExecuted() error = %v", err)
	}

	if req.Due(requestedAt.Add(60*24*time.Hour), testGrace) {
		t.Fatal("Due() after execution: want false, got true")
	}
}

func TestDeletionRequest_MarkExecuted_SetsExecutedAt(t *testing.T) {
	requestedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	executedAt := requestedAt.Add(31 * 24 * time.Hour)
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)

	if err := req.MarkExecuted(executedAt); err != nil {
		t.Fatalf("MarkExecuted() error = %v", err)
	}
	if req.ExecutedAt == nil || !req.ExecutedAt.Equal(executedAt) {
		t.Fatalf("ExecutedAt = %v, want %v", req.ExecutedAt, executedAt)
	}
}

func TestDeletionRequest_MarkExecuted_Twice_ReturnsInvalidStateError(t *testing.T) {
	requestedAt := time.Now().UTC()
	req := mustDeletionRequest(t, domain.MustNewID(), requestedAt)
	if err := req.MarkExecuted(requestedAt); err != nil {
		t.Fatalf("MarkExecuted() error = %v", err)
	}

	err := req.MarkExecuted(requestedAt.Add(time.Hour))
	if err == nil {
		t.Fatal("MarkExecuted() twice: want error, got nil")
	}
	var target *domain.InvalidStateError
	if !errors.As(err, &target) {
		t.Fatalf("second MarkExecuted() error = %T, want *domain.InvalidStateError", err)
	}
}

func TestDeletionRequest_MarshalJSON_UsesSnakeCase(t *testing.T) {
	req := mustDeletionRequest(t, domain.MustNewID(), time.Now().UTC())

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "user_id", "requested_at", "executed_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if _, ok := got["tenant_id"]; ok {
		t.Fatalf("payload %s must not contain tenant_id (AP1)", raw)
	}
}
