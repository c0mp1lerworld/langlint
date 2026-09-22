package services

import (
	"context"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

// defaultWeaknessSeverity is the neutral severity assigned to a weakness entry
// built from the error metrics. analytics does not materialize severity (it only
// lives in the WeaknessDetected event payload), and 8.4.1 forbids changing the
// MVP bounded contexts, so the on-demand endpoint uses a neutral default that the
// LLM prompt treats as a soft signal (A8, PRODUCT_DOMAIN §12.1).
const defaultWeaknessSeverity = domain.ErrorPatternSeverityModerate

// StudySessionService orchestrates the adaptive tutor (PRODUCT_DOMAIN §12.1): it
// reads the learner's aggregated weakness profile, generates the session content
// with the LLM and persists the assembled StudySession. It depends on ports only
// (A2). The slow LLM call runs outside any transaction; the save is a single
// aggregate write, so no UnitOfWork or outbox is needed.
type StudySessionService struct {
	metrics   storage.ErrorMetricRepository
	sessions  storage.StudySessionRepository
	generator ports.StudySessionGenerator
	now       func() time.Time
}

// NewStudySessionService wires the use case through its ports.
func NewStudySessionService(
	metrics storage.ErrorMetricRepository,
	sessions storage.StudySessionRepository,
	generator ports.StudySessionGenerator,
) *StudySessionService {
	return &StudySessionService{
		metrics:   metrics,
		sessions:  sessions,
		generator: generator,
		now:       time.Now,
	}
}

// Generate builds the learner's weakness profile for the window, generates the
// session content from it and persists the resulting StudySession. When there
// are no error metrics in the window it returns an InvalidStateError (409): the
// learner is not yet in a state that has detected weaknesses.
func (s *StudySessionService) Generate(ctx context.Context, userID domain.ID, window domain.Window) (*tutor.StudySession, error) {
	metrics, err := s.metrics.ListByUser(ctx, userID, window)
	if err != nil {
		return nil, err
	}

	profile, err := buildProfileFromMetrics(metrics)
	if err != nil {
		return nil, err
	}
	if len(profile.Entries) == 0 {
		return nil, &domain.InvalidStateError{Field: "weakness", Message: "no weakness detected yet"}
	}

	content, err := s.generator.Generate(ctx, ports.StudySessionRequest{Profile: profile})
	if err != nil {
		return nil, err
	}

	session, err := tutor.NewStudySession(userID, profile, content.Theory, content.Traps, content.Exercises, s.now())
	if err != nil {
		return nil, err
	}

	if err := s.sessions.Save(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// buildProfileFromMetrics converts the materialized error metrics into a
// tutor.WeaknessProfile. Because analytics does not store severity, every entry
// uses the neutral default severity; code, count and recency come from the
// aggregate (PRODUCT_DOMAIN §12.1).
func buildProfileFromMetrics(metrics []analytics.ErrorMetric) (tutor.WeaknessProfile, error) {
	entries := make([]tutor.WeaknessEntry, 0, len(metrics))
	for _, metric := range metrics {
		entry, err := tutor.NewWeaknessEntry(metric.Code, defaultWeaknessSeverity, metric.Count, metric.LastSeenAt)
		if err != nil {
			return tutor.WeaknessProfile{}, err
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return tutor.WeaknessProfile{}, nil
	}
	return tutor.NewWeaknessProfile(entries)
}
