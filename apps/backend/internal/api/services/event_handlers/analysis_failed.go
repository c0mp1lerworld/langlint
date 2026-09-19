package event_handlers

import (
	"context"
	"errors"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// AnalysisFailedHandler materializes a failed analysis in the practice status
// (analyzing -> failed). It is idempotent: it skips practices that are no
// longer analyzing.
type AnalysisFailedHandler struct {
	practices storage.PracticeRepository
	now       func() time.Time
}

// NewAnalysisFailedHandler wires the handler through its ports.
func NewAnalysisFailedHandler(practices storage.PracticeRepository) *AnalysisFailedHandler {
	return &AnalysisFailedHandler{practices: practices, now: time.Now}
}

// Handle marks the practice as failed.
func (h *AnalysisFailedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	failed, ok := analysisFailed(event)
	if !ok {
		return nil
	}

	p, err := h.practices.GetByID(ctx, failed.PracticeID)
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

	if err := p.MarkFailed(h.now()); err != nil {
		return err
	}
	return h.practices.Save(ctx, p)
}
