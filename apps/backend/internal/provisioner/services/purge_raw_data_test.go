package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/mocks"
)

func TestPurgeRawDataService_Purge_UsesRetentionCutoff(t *testing.T) {
	ctrl := gomock.NewController(t)
	now := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC)
	retention := 30 * 24 * time.Hour
	wantCutoff := now.Add(-retention)

	raw := mocks.NewMockRawDataRepository(ctrl)
	raw.EXPECT().PurgePracticesDeletedBefore(gomock.Any(), wantCutoff).Return(4, nil)

	svc := NewPurgeRawDataService(raw, retention)
	svc.now = func() time.Time { return now }

	got, err := svc.Purge(context.Background())
	if err != nil {
		t.Fatalf("Purge() error = %v", err)
	}
	if got != 4 {
		t.Fatalf("Purge() = %d, want 4", got)
	}
}

func TestPurgeRawDataService_Purge_NothingToPurge_ReturnsZero(t *testing.T) {
	ctrl := gomock.NewController(t)

	raw := mocks.NewMockRawDataRepository(ctrl)
	raw.EXPECT().PurgePracticesDeletedBefore(gomock.Any(), gomock.Any()).Return(0, nil)

	svc := NewPurgeRawDataService(raw, time.Hour)
	if got, err := svc.Purge(context.Background()); err != nil || got != 0 {
		t.Fatalf("Purge() = (%d, %v), want (0, nil)", got, err)
	}
}

func TestPurgeRawDataService_Purge_RepositoryError_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)

	raw := mocks.NewMockRawDataRepository(ctrl)
	raw.EXPECT().PurgePracticesDeletedBefore(gomock.Any(), gomock.Any()).Return(0, errors.New("delete failed"))

	svc := NewPurgeRawDataService(raw, time.Hour)
	if _, err := svc.Purge(context.Background()); err == nil {
		t.Fatal("Purge() with failing repository: want error, got nil")
	}
}
