package services

import (
	"context"
	"errors"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/events"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// practiceEventVersion is the payload version of the practice events.
const practiceEventVersion = 1

// TargetRuleInput is the raw target rule coming from the wire.
type TargetRuleInput struct {
	Verb  string
	Tense string
	Note  string
}

// PracticeInput carries the create fields of a practice.
type PracticeInput struct {
	SourceText  string
	DraftText   string
	TargetRules []TargetRuleInput
}

// PracticeUpdateInput is a partial update: a nil field is left unchanged.
type PracticeUpdateInput struct {
	SourceText  *string
	DraftText   *string
	TargetRules *[]TargetRuleInput
}

// PracticeService runs the practice use cases of the API. It depends on ports
// only (A2) and never opens a transaction itself (AP8).
type PracticeService struct {
	uow       storage.UnitOfWork
	practices storage.PracticeRepository
	analyses  storage.AnalysisRepository
	outbox    events.Outbox
	now       func() time.Time
}

// NewPracticeService wires the use cases through their ports.
func NewPracticeService(
	uow storage.UnitOfWork,
	practices storage.PracticeRepository,
	analyses storage.AnalysisRepository,
	outbox events.Outbox,
) *PracticeService {
	return &PracticeService{
		uow:       uow,
		practices: practices,
		analyses:  analyses,
		outbox:    outbox,
		now:       time.Now,
	}
}

// Create validates the input, creates the practice in draft and appends
// PracticeCreated in the same transaction (AP7).
func (s *PracticeService) Create(ctx context.Context, userID domain.ID, in PracticeInput) (*practice.Practice, error) {
	source, err := practice.NewSourceText(in.SourceText)
	if err != nil {
		return nil, err
	}
	draft, err := practice.NewDraftText(in.DraftText)
	if err != nil {
		return nil, err
	}
	rules, err := buildTargetRules(in.TargetRules)
	if err != nil {
		return nil, err
	}

	p, err := practice.NewPractice(userID, source, draft, rules, s.now())
	if err != nil {
		return nil, err
	}

	event := domain.PracticeCreated{PracticeID: p.ID, UserID: p.UserID, Version: practiceEventVersion}
	err = s.uow.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.practices.Save(txCtx, p); err != nil {
			return err
		}
		return s.outbox.Append(txCtx, event)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Get returns the user's practice and its analysis, if any. A practice owned by
// another user is reported as not found.
func (s *PracticeService) Get(ctx context.Context, userID, practiceID domain.ID) (*practice.Practice, *analysis.Analysis, error) {
	p, err := s.loadOwned(ctx, userID, practiceID)
	if err != nil {
		return nil, nil, err
	}

	a, err := s.analyses.GetByPracticeID(ctx, practiceID)
	if err != nil {
		var notFound *domain.NotFoundError
		if errors.As(err, &notFound) {
			return p, nil, nil
		}
		return nil, nil, err
	}
	return p, a, nil
}

// List returns a page of the user's practices and the total count.
func (s *PracticeService) List(ctx context.Context, userID domain.ID, limit, offset int) ([]practice.Practice, int, error) {
	return s.practices.ListByUser(ctx, userID, limit, offset)
}

// Update applies a partial edit to a draft practice.
func (s *PracticeService) Update(ctx context.Context, userID, practiceID domain.ID, in PracticeUpdateInput) (*practice.Practice, error) {
	p, err := s.loadOwned(ctx, userID, practiceID)
	if err != nil {
		return nil, err
	}

	var source *practice.SourceText
	if in.SourceText != nil {
		value, err := practice.NewSourceText(*in.SourceText)
		if err != nil {
			return nil, err
		}
		source = &value
	}

	var draft *practice.DraftText
	if in.DraftText != nil {
		value, err := practice.NewDraftText(*in.DraftText)
		if err != nil {
			return nil, err
		}
		draft = &value
	}

	var rules []practice.TargetRule
	if in.TargetRules != nil {
		rules, err = buildTargetRules(*in.TargetRules)
		if err != nil {
			return nil, err
		}
	}

	if err := p.Edit(source, draft, rules, s.now()); err != nil {
		return nil, err
	}
	if err := s.practices.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete soft-deletes the practice.
func (s *PracticeService) Delete(ctx context.Context, userID, practiceID domain.ID) error {
	p, err := s.loadOwned(ctx, userID, practiceID)
	if err != nil {
		return err
	}
	if err := p.Delete(s.now()); err != nil {
		return err
	}
	return s.practices.Save(ctx, p)
}

// Analyze transitions the practice to analyzing and appends AnalysisRequested
// atomically (AP7). Requesting an analysis that is already in progress is an
// idempotent success (AnalysisPendingError, AP4).
func (s *PracticeService) Analyze(ctx context.Context, userID, practiceID domain.ID) error {
	p, err := s.loadOwned(ctx, userID, practiceID)
	if err != nil {
		return err
	}
	if p.Status == practice.PracticeStatusAnalyzing {
		return &domain.AnalysisPendingError{Field: "status", Message: "analysis already in progress"}
	}
	if err := p.StartAnalysis(s.now()); err != nil {
		return err
	}

	event := domain.AnalysisRequested{PracticeID: p.ID, UserID: p.UserID, Version: practiceEventVersion}
	return s.uow.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.practices.Save(txCtx, p); err != nil {
			return err
		}
		return s.outbox.Append(txCtx, event)
	})
}

// loadOwned loads a practice and hides practices of other users as not found.
func (s *PracticeService) loadOwned(ctx context.Context, userID, practiceID domain.ID) (*practice.Practice, error) {
	p, err := s.practices.GetByID(ctx, practiceID)
	if err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, &domain.NotFoundError{Field: "practice", Message: "not found"}
	}
	return p, nil
}

// buildTargetRules validates and converts the raw target rules. It returns an
// empty (non-nil) slice for empty input; the aggregate enforces non-emptiness.
func buildTargetRules(inputs []TargetRuleInput) ([]practice.TargetRule, error) {
	rules := make([]practice.TargetRule, 0, len(inputs))
	for _, in := range inputs {
		rule, err := practice.NewTargetRule(in.Verb, in.Tense, in.Note)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}
