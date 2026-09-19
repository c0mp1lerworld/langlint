package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func testIdentityService(t *testing.T, email string) (*IdentityService, *mocks.MockPracticeRepository, *mocks.MockDeletionRequestRepository, *mocks.MockAccessLogRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)

	practices := mocks.NewMockPracticeRepository(ctrl)
	deletions := mocks.NewMockDeletionRequestRepository(ctrl)
	accessLog := mocks.NewMockAccessLogRepository(ctrl)

	parsed, err := identity.NewEmail(email)
	if err != nil {
		t.Fatalf("NewEmail(%q) error = %v", email, err)
	}
	return NewIdentityService(parsed, practices, deletions, accessLog), practices, deletions, accessLog
}

func TestIdentityService_Email_ReturnsConfiguredEmail(t *testing.T) {
	svc, _, _, _ := testIdentityService(t, "student@example.com")

	if svc.Email().String() != "student@example.com" {
		t.Fatalf("Email() = %q, want student@example.com", svc.Email())
	}
}

func TestIdentityService_Export_DelegatesToRepository(t *testing.T) {
	svc, practices, _, _ := testIdentityService(t, "student@example.com")
	userID := domain.MustNewID()
	want := []practice.Practice{{ID: domain.MustNewID(), UserID: userID}}
	practices.EXPECT().ListAllByUser(gomock.Any(), userID).Return(want, nil)

	got, err := svc.Export(context.Background(), userID)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Fatalf("Export() = %v, want %v", got, want)
	}
}

func TestIdentityService_Export_RepositoryError_ReturnsError(t *testing.T) {
	svc, practices, _, _ := testIdentityService(t, "student@example.com")
	userID := domain.MustNewID()
	practices.EXPECT().ListAllByUser(gomock.Any(), userID).Return(nil, errors.New("db down"))

	if _, err := svc.Export(context.Background(), userID); err == nil {
		t.Fatal("Export() with failing repository: want error, got nil")
	}
}

func TestIdentityService_RequestDeletion_NoPending_AppendsRequest(t *testing.T) {
	svc, _, deletions, _ := testIdentityService(t, "student@example.com")
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	deletions.EXPECT().HasPending(gomock.Any(), userID).Return(false, nil)
	deletions.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *identity.DeletionRequest) error {
			if req.UserID != userID {
				t.Fatalf("Append() user = %s, want %s", req.UserID, userID)
			}
			if !req.RequestedAt.Equal(now) {
				t.Fatalf("Append() requested_at = %v, want %v", req.RequestedAt, now)
			}
			if req.ExecutedAt != nil {
				t.Fatalf("Append() executed_at = %v, want nil", req.ExecutedAt)
			}
			return nil
		},
	)

	if err := svc.RequestDeletion(context.Background(), userID); err != nil {
		t.Fatalf("RequestDeletion() error = %v", err)
	}
}

func TestIdentityService_RequestDeletion_AlreadyPending_IsIdempotent(t *testing.T) {
	svc, _, deletions, _ := testIdentityService(t, "student@example.com")
	userID := domain.MustNewID()
	deletions.EXPECT().HasPending(gomock.Any(), userID).Return(true, nil)
	// No Append expected: the call is idempotent.

	if err := svc.RequestDeletion(context.Background(), userID); err != nil {
		t.Fatalf("RequestDeletion() error = %v", err)
	}
}

func TestIdentityService_RequestDeletion_HasPendingError_ReturnsError(t *testing.T) {
	svc, _, deletions, _ := testIdentityService(t, "student@example.com")
	userID := domain.MustNewID()
	deletions.EXPECT().HasPending(gomock.Any(), userID).Return(false, errors.New("db down"))

	if err := svc.RequestDeletion(context.Background(), userID); err == nil {
		t.Fatal("RequestDeletion() with failing HasPending: want error, got nil")
	}
}

func TestIdentityService_AccessLog_DelegatesToRepository(t *testing.T) {
	svc, _, _, accessLog := testIdentityService(t, "student@example.com")
	userID := domain.MustNewID()
	event, err := identity.NewAccessEvent(userID, "GET", "/practices", "", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewAccessEvent() error = %v", err)
	}
	accessLog.EXPECT().ListByUser(gomock.Any(), userID, 20, 0).Return([]identity.AccessEvent{*event}, 1, nil)

	items, total, err := svc.AccessLog(context.Background(), userID, 20, 0)
	if err != nil {
		t.Fatalf("AccessLog() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != event.ID {
		t.Fatalf("AccessLog() = (%v, %d), want one event", items, total)
	}
}
