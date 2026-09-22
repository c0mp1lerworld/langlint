package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func TestStudySessionService_Generate_WithMetrics_ReturnsSavedSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).
		Return([]analytics.ErrorMetric{
			{Code: domain.ErrorPatternCodeWordOrder, Count: 6, LastSeenAt: now},
		}, nil)

	generator := mocks.NewMockStudySessionGenerator(ctrl)
	generator.EXPECT().Generate(gomock.Any(), gomock.Any()).
		Return(ports.StudySessionContent{
			Theory: "El orden de palabras en inglés es SVO.",
			Traps:  []tutor.Trap{{Code: domain.ErrorPatternCodeWordOrder, Description: "Colocar el verbo al final."}},
			Exercises: []tutor.Exercise{
				{Kind: tutor.ExerciseKindFill, Prompt: "I ___ (never) have seen.", Answer: "have never"},
			},
		}, nil)

	sessions := mocks.NewMockStudySessionRepository(ctrl)
	sessions.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	svc := NewStudySessionService(metrics, sessions, generator)
	session, err := svc.Generate(context.Background(), userID, domain.WindowWeek)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if session.UserID != userID {
		t.Fatalf("UserID = %s, want %s", session.UserID, userID)
	}
	if session.Status != tutor.StudySessionStatusGenerated {
		t.Fatalf("Status = %q, want generated", session.Status)
	}
	if session.Theory == "" || len(session.Traps) != 1 || len(session.Exercises) != 1 {
		t.Fatalf("session content = theory=%q traps=%d exercises=%d", session.Theory, len(session.Traps), len(session.Exercises))
	}
	if len(session.Profile.Entries) != 1 {
		t.Fatalf("Profile.Entries len = %d, want 1", len(session.Profile.Entries))
	}
	if session.Profile.Entries[0].Severity != domain.ErrorPatternSeverityModerate {
		t.Fatalf("Profile severity = %q, want moderate", session.Profile.Entries[0].Severity)
	}
}

func TestStudySessionService_Generate_NoMetrics_ReturnsInvalidState(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).Return(nil, nil)

	svc := NewStudySessionService(metrics, mocks.NewMockStudySessionRepository(ctrl), mocks.NewMockStudySessionGenerator(ctrl))
	if _, err := svc.Generate(context.Background(), userID, domain.WindowWeek); err == nil {
		t.Fatal("Generate() error = nil, want InvalidStateError")
	} else if _, ok := err.(*domain.InvalidStateError); !ok {
		t.Fatalf("Generate() error = %T, want *domain.InvalidStateError", err)
	}
}

func TestStudySessionService_Generate_GeneratorError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, domain.WindowWeek).
		Return([]analytics.ErrorMetric{{Code: domain.ErrorPatternCodeFalseFriend, Count: 2, LastSeenAt: now}}, nil)

	generator := mocks.NewMockStudySessionGenerator(ctrl)
	generator.EXPECT().Generate(gomock.Any(), gomock.Any()).
		Return(ports.StudySessionContent{}, &domain.LLMUnavailableError{Message: "llm unavailable"})

	svc := NewStudySessionService(metrics, mocks.NewMockStudySessionRepository(ctrl), generator)
	if _, err := svc.Generate(context.Background(), userID, domain.WindowWeek); err == nil {
		t.Fatal("Generate() error = nil, want LLMUnavailableError")
	}
}

func TestBuildProfileFromMetrics_AssignsDefaultSeverityAndOrdersByFrequency(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	metrics := []analytics.ErrorMetric{
		{Code: domain.ErrorPatternCodeFalseFriend, Count: 2, LastSeenAt: now},
		{Code: domain.ErrorPatternCodeWordOrder, Count: 6, LastSeenAt: now},
	}

	profile, err := buildProfileFromMetrics(metrics)
	if err != nil {
		t.Fatalf("buildProfileFromMetrics() error = %v", err)
	}
	if len(profile.Entries) != 2 {
		t.Fatalf("Entries len = %d, want 2", len(profile.Entries))
	}
	if profile.Weakest().Code != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("Weakest().Code = %q, want word_order", profile.Weakest().Code)
	}
	for _, entry := range profile.Entries {
		if entry.Severity != domain.ErrorPatternSeverityModerate {
			t.Fatalf("entry %q severity = %q, want moderate", entry.Code, entry.Severity)
		}
	}
}
