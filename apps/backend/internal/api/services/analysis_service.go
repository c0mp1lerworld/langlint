// Package services holds the application use cases of the API entry point (A2).
// Services depend on ports only and never on adapters or infrastructure.
package services

import (
	"context"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// analysisEventVersion is the payload version of AnalysisCompleted (§5.1).
const analysisEventVersion = 1

// AnalysisService orchestrates the cognitive analysis of a practice
// (PRODUCT_DOMAIN §4.7). The slow LLM call runs outside the transaction (4.2.3)
// and its result is persisted together with the outbox event atomically (4.2.4).
type AnalysisService struct {
	extractor ports.LLMExtractor
	uow       storage.UnitOfWork
	practices storage.PracticeRepository
	analyses  storage.AnalysisRepository
	outbox    events.Outbox
	now       func() time.Time
}

// NewAnalysisService wires the use case through its ports (A2). The service
// never sees a database driver, a transaction handle or the provider SDK.
func NewAnalysisService(
	extractor ports.LLMExtractor,
	uow storage.UnitOfWork,
	practices storage.PracticeRepository,
	analyses storage.AnalysisRepository,
	outbox events.Outbox,
) *AnalysisService {
	return &AnalysisService{
		extractor: extractor,
		uow:       uow,
		practices: practices,
		analyses:  analyses,
		outbox:    outbox,
		now:       time.Now,
	}
}

// RunAnalysis extracts the fragment analysis of a practice and persists the
// completed Analysis with its AnalysisCompleted outbox event. The LLM call
// happens before InTransaction so the slow network operation never holds a
// database connection (AP7, AP8); the Analysis and the event commit atomically
// so analytics never observes a completion whose event was lost (4.2.3/4.2.4).
//
// The failure path (persisting a failed Analysis and emitting AnalysisFailed)
// belongs to a later item; for now a provider error propagates untouched.
func (s *AnalysisService) RunAnalysis(ctx context.Context, practiceID domain.ID) error {
	p, err := s.practices.GetByID(ctx, practiceID)
	if err != nil {
		return err
	}

	fragments, err := s.extractor.Extract(ctx, ports.ExtractRequest{
		PracticeID:  practiceID,
		SourceText:  p.SourceText.String(),
		DraftText:   p.DraftText.String(),
		TargetRules: p.TargetRules,
	})
	if err != nil {
		return err
	}

	result, err := analysis.NewAnalysis(practiceID, s.extractor.Model(), s.extractor.ModelVersion(), s.now())
	if err != nil {
		return err
	}
	if err := result.Complete(fragments); err != nil {
		return err
	}

	event := domain.AnalysisCompleted{
		AnalysisID:    result.ID,
		PracticeID:    practiceID,
		UserID:        p.UserID,
		ErrorPatterns: collectErrorPatterns(fragments),
		Version:       analysisEventVersion,
	}

	return s.uow.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.analyses.Save(txCtx, result); err != nil {
			return err
		}
		return s.outbox.Append(txCtx, event)
	})
}

// collectErrorPatterns flattens the error patterns of every fragment into the
// payload of AnalysisCompleted (PRODUCT_DOMAIN §4.4). It returns an empty slice,
// never nil, so the event serializes as an array.
func collectErrorPatterns(fragments []analysis.Fragment) []domain.ErrorPattern {
	patterns := []domain.ErrorPattern{}
	for _, fragment := range fragments {
		patterns = append(patterns, fragment.ErrorPatterns...)
	}
	return patterns
}
