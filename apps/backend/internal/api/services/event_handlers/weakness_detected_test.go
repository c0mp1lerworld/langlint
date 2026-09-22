package event_handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestWeaknessDetectedHandler_ValidEvent_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).
		Return([]analytics.ErrorMetric{{Code: domain.ErrorPatternCodeWordOrder, Count: 6, LastSeenAt: now}}, nil)

	h := NewWeaknessDetectedHandler(metrics)
	event := domain.WeaknessDetected{
		UserID:        userID,
		Window:        domain.WindowWeek,
		ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityModerate}},
	}
	if err := h.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestWeaknessDetectedHandler_WrongEvent_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewWeaknessDetectedHandler(mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), domain.PracticeCreated{}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}

func TestWeaknessDetectedHandler_PointerEvent_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).
		Return([]analytics.ErrorMetric{{Code: domain.ErrorPatternCodeWordOrder, Count: 6, LastSeenAt: now}}, nil)

	h := NewWeaknessDetectedHandler(metrics)
	event := &domain.WeaknessDetected{
		UserID:        userID,
		Window:        domain.WindowWeek,
		ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityModerate}},
	}
	if err := h.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestWeaknessDetectedHandler_MetricsError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).Return(nil, errors.New("db down"))

	h := NewWeaknessDetectedHandler(metrics)
	if err := h.Handle(context.Background(), domain.WeaknessDetected{UserID: userID, Window: domain.WindowWeek}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestBuildWeaknessProfile_EnrichesDetectedPatterns(t *testing.T) {
	now := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)
	userID := domain.MustNewID()

	metrics := []analytics.ErrorMetric{
		{UserID: userID, Code: domain.ErrorPatternCodeWordOrder, Window: analytics.WindowWeek, Count: 6, LastSeenAt: now},
		{UserID: userID, Code: domain.ErrorPatternCodeFalseFriend, Window: analytics.WindowWeek, Count: 2, LastSeenAt: now},
	}
	patterns := []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityModerate},
		{Code: domain.ErrorPatternCodeFalseFriend, Severity: domain.ErrorPatternSeverityMinor},
	}

	profile, err := buildWeaknessProfile(metrics, patterns)
	if err != nil {
		t.Fatalf("buildWeaknessProfile() error = %v", err)
	}
	if len(profile.Entries) != 2 {
		t.Fatalf("Entries len = %d, want 2", len(profile.Entries))
	}
	if profile.Weakest().Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Weakest().Code = %q, want word_order", profile.Weakest().Code)
	}
	for _, entry := range profile.Entries {
		switch entry.Code {
		case domain.ErrorPatternCodeWordOrder:
			if entry.Severity != domain.ErrorPatternSeverityModerate || entry.Count != 6 {
				t.Fatalf("word_order entry = %+v, want severity moderate count 6", entry)
			}
		case domain.ErrorPatternCodeFalseFriend:
			if entry.Severity != domain.ErrorPatternSeverityMinor || entry.Count != 2 {
				t.Fatalf("false_friend entry = %+v, want severity minor count 2", entry)
			}
		default:
			t.Fatalf("unexpected entry code %q", entry.Code)
		}
	}
}

func TestBuildWeaknessProfile_PatternWithoutMetric_IsSkipped(t *testing.T) {
	now := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

	metrics := []analytics.ErrorMetric{
		{Code: domain.ErrorPatternCodeWordOrder, Window: analytics.WindowWeek, Count: 6, LastSeenAt: now},
	}
	patterns := []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityModerate},
		{Code: domain.ErrorPatternCodeFalseFriend, Severity: domain.ErrorPatternSeverityMinor},
	}

	profile, err := buildWeaknessProfile(metrics, patterns)
	if err != nil {
		t.Fatalf("buildWeaknessProfile() error = %v", err)
	}
	if len(profile.Entries) != 1 {
		t.Fatalf("Entries len = %d, want 1", len(profile.Entries))
	}
	if profile.Entries[0].Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Entries[0].Code = %q, want word_order", profile.Entries[0].Code)
	}
}

func TestBuildWeaknessProfile_NoMetrics_ReturnsEmptyProfile(t *testing.T) {
	profile, err := buildWeaknessProfile(nil, []domain.ErrorPattern{{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityModerate}})
	if err != nil {
		t.Fatalf("buildWeaknessProfile() error = %v", err)
	}
	if len(profile.Entries) != 0 {
		t.Fatalf("Entries len = %d, want 0", len(profile.Entries))
	}
}

func TestBuildWeaknessProfile_InvalidSeverity_ReturnsError(t *testing.T) {
	now := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

	metrics := []analytics.ErrorMetric{
		{Code: domain.ErrorPatternCodeWordOrder, Window: analytics.WindowWeek, Count: 6, LastSeenAt: now},
	}
	patterns := []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeWordOrder, Severity: "fatal"},
	}

	if _, err := buildWeaknessProfile(metrics, patterns); err == nil {
		t.Fatal("buildWeaknessProfile() error = nil, want error")
	}
}
