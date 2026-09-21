// Package services holds the application use cases of the API entry point (A2).
// Services depend on ports only and never on adapters or infrastructure.
package services

import (
	"context"
	"errors"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

const (
	// analysisEventVersion is the payload version of the analysis events (§5.1).
	analysisEventVersion = 1

	// defaultLLMTimeout bounds the slow external LLM call (4.3.3) when no
	// APP_LLM_TIMEOUT is configured. The exhaustive feedback (several structured
	// entries per fragment) can take well over a minute on long texts.
	defaultLLMTimeout = 180 * time.Second

	// Generic failure reasons stored in AnalysisFailed. They never carry the raw
	// provider error (A5, A8).
	reasonLLMUnavailable     = "llm unavailable"
	reasonLLMOutputTruncated = "llm output truncated"
	reasonInvalidResult      = "invalid result"
)

// LLMTimeout is the maximum duration of a single LLM extraction call. It is a
// named type so Fx can inject it without ambiguity with other durations (A2).
type LLMTimeout time.Duration

// AnalysisService orchestrates the cognitive analysis of a practice
// (PRODUCT_DOMAIN §4.7). The slow LLM call runs outside the transaction (4.2.3)
// with a timeout (4.3.3) and its result is persisted together with the outbox
// event atomically (4.2.4).
type AnalysisService struct {
	extractor  ports.LLMExtractor
	uow        storage.UnitOfWork
	practices  storage.PracticeRepository
	analyses   storage.AnalysisRepository
	outbox     events.Outbox
	now        func() time.Time
	llmTimeout time.Duration
}

// NewAnalysisService wires the use case through its ports (A2). The service
// never sees a database driver, a transaction handle or the provider SDK.
func NewAnalysisService(
	extractor ports.LLMExtractor,
	uow storage.UnitOfWork,
	practices storage.PracticeRepository,
	analyses storage.AnalysisRepository,
	outbox events.Outbox,
	llmTimeout LLMTimeout,
) *AnalysisService {
	timeout := time.Duration(llmTimeout)
	if timeout <= 0 {
		timeout = defaultLLMTimeout
	}
	return &AnalysisService{
		extractor:  extractor,
		uow:        uow,
		practices:  practices,
		analyses:   analyses,
		outbox:     outbox,
		now:        time.Now,
		llmTimeout: timeout,
	}
}

// RunAnalysis extracts the fragment analysis of a practice and persists either
// the completed Analysis with AnalysisCompleted, or a failed Analysis with
// AnalysisFailed (4.2.3/4.2.4/4.3.3). The practice text is anonymized before it
// leaves the process (A8) and the LLM call happens before InTransaction so the
// slow network operation never holds a database connection (AP7, AP8).
//
// A provider or result failure is handled (not propagated): it is recorded as
// analysis.status == failed together with the AnalysisFailed event, so the
// event relay never retries a terminal failure. The raw provider error never
// reaches the event or the caller.
func (s *AnalysisService) RunAnalysis(ctx context.Context, practiceID domain.ID) error {
	p, err := s.practices.GetByID(ctx, practiceID)
	if err != nil {
		return err
	}

	result, err := analysis.NewAnalysis(practiceID, s.extractor.Model(), s.extractor.ModelVersion(), s.now())
	if err != nil {
		return err
	}

	llmCtx, cancel := context.WithTimeout(ctx, s.llmTimeout)
	defer cancel()

	fragments, err := s.extractor.Extract(llmCtx, ports.ExtractRequest{
		PracticeID:  practiceID,
		SourceText:  AnonymizeSpanish(p.SourceText.String()),
		DraftText:   Anonymize(p.DraftText.String()),
		TargetRules: p.TargetRules,
	})
	if err != nil {
		var truncated *domain.LLMOutputTruncatedError
		if errors.As(err, &truncated) {
			return s.failAnalysis(ctx, result, reasonLLMOutputTruncated)
		}
		return s.failAnalysis(ctx, result, reasonLLMUnavailable)
	}

	if err := result.Complete(fragments); err != nil {
		return s.failAnalysis(ctx, result, reasonInvalidResult)
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

// failAnalysis records a handled analysis failure: it transitions the pending
// analysis to failed and persists it together with the AnalysisFailed event in
// one transaction (4.3.3). The reason is a generic constant, never the raw
// provider error (A5, A8).
func (s *AnalysisService) failAnalysis(ctx context.Context, result *analysis.Analysis, reason string) error {
	if err := result.Fail(); err != nil {
		return err
	}

	event := domain.AnalysisFailed{
		AnalysisID: result.ID,
		PracticeID: result.PracticeID,
		Reason:     reason,
		Version:    analysisEventVersion,
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
