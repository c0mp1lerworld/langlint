package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/mocks"
)

const testDeletionGrace = 30 * 24 * time.Hour

func pendingRequest(t *testing.T, userID domain.ID, requestedAt time.Time) identity.DeletionRequest {
	t.Helper()
	req, err := identity.NewDeletionRequest(userID, requestedAt)
	if err != nil {
		t.Fatalf("NewDeletionRequest() error = %v", err)
	}
	return *req
}

func TestExecuteDeletionsService_Run_ExecutesOnlyDueRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	now := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC)

	due := pendingRequest(t, domain.MustNewID(), now.Add(-31*24*time.Hour))
	notDue := pendingRequest(t, domain.MustNewID(), now.Add(-29*24*time.Hour))

	deletions := mocks.NewMockDeletionRepository(ctrl)
	deletions.EXPECT().ListPending(gomock.Any()).Return([]identity.DeletionRequest{due, notDue}, nil)
	deletions.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *identity.DeletionRequest) error {
			if req.ID != due.ID {
				t.Fatalf("Execute() got request %s, want the due one %s", req.ID, due.ID)
			}
			if req.ExecutedAt == nil || !req.ExecutedAt.Equal(now) {
				t.Fatalf("Execute() request executed_at = %v, want %v", req.ExecutedAt, now)
			}
			return nil
		},
	)

	svc := NewExecuteDeletionsService(deletions, testDeletionGrace)
	svc.now = func() time.Time { return now }

	got, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got != 1 {
		t.Fatalf("Run() = %d, want 1", got)
	}
}

func TestExecuteDeletionsService_Run_NoPending_ReturnsZero(t *testing.T) {
	ctrl := gomock.NewController(t)

	deletions := mocks.NewMockDeletionRepository(ctrl)
	deletions.EXPECT().ListPending(gomock.Any()).Return(nil, nil)

	svc := NewExecuteDeletionsService(deletions, testDeletionGrace)
	if got, err := svc.Run(context.Background()); err != nil || got != 0 {
		t.Fatalf("Run() = (%d, %v), want (0, nil)", got, err)
	}
}

func TestExecuteDeletionsService_Run_ListError_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)

	deletions := mocks.NewMockDeletionRepository(ctrl)
	deletions.EXPECT().ListPending(gomock.Any()).Return(nil, errors.New("list failed"))

	svc := NewExecuteDeletionsService(deletions, testDeletionGrace)
	if _, err := svc.Run(context.Background()); err == nil {
		t.Fatal("Run() with failing ListPending: want error, got nil")
	}
}

func TestExecuteDeletionsService_Run_ExecuteError_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	now := time.Now().UTC()

	due := pendingRequest(t, domain.MustNewID(), now.Add(-31*24*time.Hour))

	deletions := mocks.NewMockDeletionRepository(ctrl)
	deletions.EXPECT().ListPending(gomock.Any()).Return([]identity.DeletionRequest{due}, nil)
	deletions.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(errors.New("delete failed"))

	svc := NewExecuteDeletionsService(deletions, testDeletionGrace)
	svc.now = func() time.Time { return now }

	if _, err := svc.Run(context.Background()); err == nil {
		t.Fatal("Run() with failing Execute: want error, got nil")
	}
}
