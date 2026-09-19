package event_handlers

import (
	"context"
	"errors"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// AnalysisRunner runs the cognitive analysis of a practice. It is satisfied by
// services.AnalysisService; the narrow interface keeps the handler decoupled
// and testable (A2).
type AnalysisRunner interface {
	RunAnalysis(ctx context.Context, practiceID domain.ID) error
}

// AnalysisRequestedHandler triggers the cognitive analysis when the user asks
// for it. It is idempotent: it skips practices that are no longer analyzing or
// that already have an analysis (at-least-once delivery, manifest §5.1).
type AnalysisRequestedHandler struct {
	practices storage.PracticeRepository
	analyses  storage.AnalysisRepository
	runner    AnalysisRunner
}

// NewAnalysisRequestedHandler wires the handler through its ports.
func NewAnalysisRequestedHandler(
	practices storage.PracticeRepository,
	analyses storage.AnalysisRepository,
	runner AnalysisRunner,
) *AnalysisRequestedHandler {
	return &AnalysisRequestedHandler{practices: practices, analyses: analyses, runner: runner}
}

// Handle runs the analysis of the requested practice.
func (h *AnalysisRequestedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	requested, ok := analysisRequested(event)
	if !ok {
		return nil
	}

	p, err := h.practices.GetByID(ctx, requested.PracticeID)
	if err != nil {
		var notFound *domain.NotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return err
	}
	if p.Status != practice.PracticeStatusAnalyzing {
		return nil
	}

	if _, err := h.analyses.GetByPracticeID(ctx, requested.PracticeID); err == nil {
		return nil
	} else {
		var notFound *domain.NotFoundError
		if !errors.As(err, &notFound) {
			return err
		}
	}

	return h.runner.RunAnalysis(ctx, requested.PracticeID)
}
