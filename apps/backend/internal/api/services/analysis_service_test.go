package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// txMarker tags the context handed to the UnitOfWork closure so tests can prove
// which calls run inside the transaction and which run outside it (4.2.3).
type txMarker struct{}

func testPractice() *practice.Practice {
	return &practice.Practice{
		ID:          domain.MustNewID(),
		UserID:      domain.MustNewID(),
		SourceText:  practice.SourceText("El perro corre."),
		DraftText:   practice.DraftText("The dog run."),
		TargetRules: []practice.TargetRule{{Verb: "run", Tense: "past simple"}},
		Status:      practice.PracticeStatusAnalyzing,
	}
}

func testFragments() []analysis.Fragment {
	return []analysis.Fragment{
		{
			SourceES:   "El perro corre.",
			UserDraft:  "The dog run.",
			Correction: "The dog runs.",
			TargetVerbReviews: []analysis.TargetVerbReview{{
				Verb:         "run",
				CorrectForm:  "runs",
				Rule:         "tercera persona singular",
				Why:          "el sujeto es singular",
				ESContrast:   "en español no cambia",
				Alternatives: []string{"runs"},
			}},
			LexicalClarifications: []analysis.LexicalClarification{{
				Term:     "run",
				Meaning:  "correr",
				WhyWrong: "falta la -s de tercera persona",
			}},
			GrammarExplanations: []analysis.GrammarExplanation{{
				RuleName:       "tercera persona singular",
				Explanation:    "el verbo añade -s",
				Construction:   "verbo + -s",
				Counterexample: "run -> runs",
				Exception:      "verbos irregulares",
				ESContrast:     "no aplica en español",
			}},
			ErrorPatterns: []domain.ErrorPattern{
				{
					Code:     domain.ErrorPatternCodeInfinitiveConjugation,
					Severity: domain.ErrorPatternSeverityModerate,
					Note:     "missing third person -s",
				},
			},
		},
	}
}

type serviceFixture struct {
	svc       *AnalysisService
	extractor *mocks.MockLLMExtractor
	uow       *mocks.MockUnitOfWork
	practices *mocks.MockPracticeRepository
	analyses  *mocks.MockAnalysisRepository
	outbox    *mocks.MockOutbox
	now       time.Time
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	fixture := &serviceFixture{
		extractor: mocks.NewMockLLMExtractor(ctrl),
		uow:       mocks.NewMockUnitOfWork(ctrl),
		practices: mocks.NewMockPracticeRepository(ctrl),
		analyses:  mocks.NewMockAnalysisRepository(ctrl),
		outbox:    mocks.NewMockOutbox(ctrl),
		now:       time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC),
	}
	fixture.svc = NewAnalysisService(fixture.extractor, fixture.uow, fixture.practices, fixture.analyses, fixture.outbox)
	fixture.svc.now = func() time.Time { return fixture.now }
	return fixture
}

// expectTransaction runs the UnitOfWork closure with a marked context.
func (f *serviceFixture) expectTransaction() {
	f.uow.EXPECT().InTransaction(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(context.WithValue(ctx, txMarker{}, true))
		},
	)
}

func TestAnalysisService_RunAnalysis_Success_PersistsAnalysisAndEventAtomically(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()
	fragments := testFragments()

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Extract(gomock.Any(), ports.ExtractRequest{
		PracticeID:  p.ID,
		SourceText:  p.SourceText.String(),
		DraftText:   p.DraftText.String(),
		TargetRules: p.TargetRules,
	}).Return(fragments, nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.expectTransaction()

	var saved *analysis.Analysis
	var dispatched domain.DomainEvent
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, result *analysis.Analysis) error {
			saved = result
			return nil
		},
	)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event domain.DomainEvent) error {
			dispatched = event
			return nil
		},
	)

	if err := fixture.svc.RunAnalysis(context.Background(), p.ID); err != nil {
		t.Fatalf("RunAnalysis() error = %v, want nil", err)
	}

	if saved == nil {
		t.Fatal("Analysis was not saved")
	}
	if saved.Status != analysis.AnalysisStatusCompleted {
		t.Fatalf("saved.Status = %q, want completed", saved.Status)
	}
	if len(saved.Fragments) != 1 || saved.Fragments[0].Correction != fragments[0].Correction {
		t.Fatalf("saved.Fragments = %v, want %v", saved.Fragments, fragments)
	}
	if saved.Model != "gpt-4o-mini" || saved.ModelVersion != "2024-07-18" {
		t.Fatalf("saved model = %q/%q, want gpt-4o-mini/2024-07-18", saved.Model, saved.ModelVersion)
	}
	if !saved.CreatedAt.Equal(fixture.now) {
		t.Fatalf("saved.CreatedAt = %v, want %v", saved.CreatedAt, fixture.now)
	}

	event, ok := dispatched.(domain.AnalysisCompleted)
	if !ok {
		t.Fatalf("dispatched event = %T, want domain.AnalysisCompleted", dispatched)
	}
	if event.AnalysisID != saved.ID || event.PracticeID != p.ID || event.UserID != p.UserID {
		t.Fatalf("event ids = %+v, want analysis %v practice %v user %v", event, saved.ID, p.ID, p.UserID)
	}
	if event.Version != analysisEventVersion {
		t.Fatalf("event.Version = %d, want %d", event.Version, analysisEventVersion)
	}
	if len(event.ErrorPatterns) != 1 || event.ErrorPatterns[0] != fragments[0].ErrorPatterns[0] {
		t.Fatalf("event.ErrorPatterns = %v, want %v", event.ErrorPatterns, fragments[0].ErrorPatterns)
	}
}

func TestAnalysisService_RunAnalysis_ExtractRunsOutsideTransaction(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()
	fragments := testFragments()

	var extractCtx, saveCtx, appendCtx context.Context

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ ports.ExtractRequest) ([]analysis.Fragment, error) {
			extractCtx = ctx
			return fragments, nil
		},
	)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.expectTransaction()
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ *analysis.Analysis) error {
			saveCtx = ctx
			return nil
		},
	)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ domain.DomainEvent) error {
			appendCtx = ctx
			return nil
		},
	)

	if err := fixture.svc.RunAnalysis(context.Background(), p.ID); err != nil {
		t.Fatalf("RunAnalysis() error = %v, want nil", err)
	}

	if extractCtx.Value(txMarker{}) != nil {
		t.Fatal("Extract ran inside the transaction, want outside (4.2.3)")
	}
	if saveCtx == nil || saveCtx.Value(txMarker{}) == nil {
		t.Fatal("Save ran outside the transaction, want inside (4.2.4)")
	}
	if appendCtx == nil || appendCtx.Value(txMarker{}) == nil {
		t.Fatal("Append ran outside the transaction, want inside (4.2.4)")
	}
}

func TestAnalysisService_RunAnalysis_PracticeNotFound_ReturnsErrorWithoutCallingLLM(t *testing.T) {
	fixture := newServiceFixture(t)
	practiceID := domain.MustNewID()

	fixture.practices.EXPECT().GetByID(gomock.Any(), practiceID).Return(nil, &domain.NotFoundError{Field: "practice_id", Message: "not found"})

	err := fixture.svc.RunAnalysis(context.Background(), practiceID)
	var target *domain.NotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("RunAnalysis() error = %v, want *domain.NotFoundError", err)
	}
}

func TestAnalysisService_RunAnalysis_ExtractorError_PersistsFailedAnalysisAndEmitsAnalysisFailed(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).Return(nil, &domain.LLMUnavailableError{Message: "llm unavailable"})
	fixture.expectTransaction()

	var saved *analysis.Analysis
	var dispatched domain.DomainEvent
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, result *analysis.Analysis) error {
			saved = result
			return nil
		},
	)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event domain.DomainEvent) error {
			dispatched = event
			return nil
		},
	)

	if err := fixture.svc.RunAnalysis(context.Background(), p.ID); err != nil {
		t.Fatalf("RunAnalysis() error = %v, want nil (handled failure)", err)
	}
	if saved == nil {
		t.Fatal("failed Analysis was not saved")
	}
	if saved.Status != analysis.AnalysisStatusFailed {
		t.Fatalf("saved.Status = %q, want failed", saved.Status)
	}

	event, ok := dispatched.(domain.AnalysisFailed)
	if !ok {
		t.Fatalf("dispatched event = %T, want domain.AnalysisFailed", dispatched)
	}
	if event.AnalysisID != saved.ID || event.PracticeID != p.ID {
		t.Fatalf("event ids = %+v, want analysis %v practice %v", event, saved.ID, p.ID)
	}
	if event.Reason != reasonLLMUnavailable {
		t.Fatalf("event.Reason = %q, want %q", event.Reason, reasonLLMUnavailable)
	}
	if event.Version != analysisEventVersion {
		t.Fatalf("event.Version = %d, want %d", event.Version, analysisEventVersion)
	}
}

func TestAnalysisService_RunAnalysis_EmptyFragments_PersistsFailedAnalysis(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).Return([]analysis.Fragment{}, nil)
	fixture.expectTransaction()

	var saved *analysis.Analysis
	var dispatched domain.DomainEvent
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, result *analysis.Analysis) error {
			saved = result
			return nil
		},
	)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event domain.DomainEvent) error {
			dispatched = event
			return nil
		},
	)

	if err := fixture.svc.RunAnalysis(context.Background(), p.ID); err != nil {
		t.Fatalf("RunAnalysis() error = %v, want nil (handled failure)", err)
	}

	event, ok := dispatched.(domain.AnalysisFailed)
	if !ok {
		t.Fatalf("dispatched event = %T, want domain.AnalysisFailed", dispatched)
	}
	if event.Reason != reasonInvalidResult {
		t.Fatalf("event.Reason = %q, want %q", event.Reason, reasonInvalidResult)
	}
	if saved == nil || saved.Status != analysis.AnalysisStatusFailed {
		t.Fatalf("saved.Status = %v, want failed", saved)
	}
}

func TestAnalysisService_RunAnalysis_AnonymizesInputBeforeLLM(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()
	p.SourceText = practice.SourceText("Ayer María me escribió a maria@example.com desde Barcelona.")
	p.DraftText = practice.DraftText("Yesterday Maria wrote me at maria@example.com from Barcelona.")

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.extractor.EXPECT().Extract(gomock.Any(), ports.ExtractRequest{
		PracticeID:  p.ID,
		SourceText:  "Ayer [name] me escribió a [email] desde [name].",
		DraftText:   "Yesterday [name] wrote me at [email] from Barcelona.",
		TargetRules: p.TargetRules,
	}).Return(testFragments(), nil)
	fixture.expectTransaction()
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	if err := fixture.svc.RunAnalysis(context.Background(), p.ID); err != nil {
		t.Fatalf("RunAnalysis() error = %v, want nil", err)
	}
}

func TestAnalysisService_RunAnalysis_ConfiguresLLMDeadline(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()
	fixture.svc.llmTimeout = 5 * time.Second

	var extractCtx context.Context
	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ ports.ExtractRequest) ([]analysis.Fragment, error) {
			extractCtx = ctx
			return testFragments(), nil
		},
	)
	fixture.expectTransaction()
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	if err := fixture.svc.RunAnalysis(context.Background(), p.ID); err != nil {
		t.Fatalf("RunAnalysis() error = %v, want nil", err)
	}

	deadline, ok := extractCtx.Deadline()
	if !ok {
		t.Fatal("Extract context has no deadline, want one (4.3.3)")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 5*time.Second {
		t.Fatalf("deadline remaining = %v, want within (0, 5s]", remaining)
	}
}

func TestAnalysisService_RunAnalysis_FailurePersistFails_ReturnsError(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).Return(nil, &domain.LLMUnavailableError{Message: "llm unavailable"})
	fixture.expectTransaction()
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).Return(&domain.InternalError{Message: "internal"})

	err := fixture.svc.RunAnalysis(context.Background(), p.ID)
	var target *domain.InternalError
	if !errors.As(err, &target) {
		t.Fatalf("RunAnalysis() error = %v, want *domain.InternalError", err)
	}
}

func TestAnalysisService_RunAnalysis_SaveFails_ReturnsErrorWithoutAppending(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).Return(testFragments(), nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.expectTransaction()
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).Return(&domain.InternalError{Message: "internal"})

	err := fixture.svc.RunAnalysis(context.Background(), p.ID)
	var target *domain.InternalError
	if !errors.As(err, &target) {
		t.Fatalf("RunAnalysis() error = %v, want *domain.InternalError", err)
	}
}

func TestAnalysisService_RunAnalysis_AppendFails_ReturnsError(t *testing.T) {
	fixture := newServiceFixture(t)
	p := testPractice()

	fixture.practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	fixture.extractor.EXPECT().Extract(gomock.Any(), gomock.Any()).Return(testFragments(), nil)
	fixture.extractor.EXPECT().Model().Return("gpt-4o-mini")
	fixture.extractor.EXPECT().ModelVersion().Return("2024-07-18")
	fixture.expectTransaction()
	fixture.analyses.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	fixture.outbox.EXPECT().Append(gomock.Any(), gomock.Any()).Return(&domain.InternalError{Message: "internal"})

	err := fixture.svc.RunAnalysis(context.Background(), p.ID)
	var target *domain.InternalError
	if !errors.As(err, &target) {
		t.Fatalf("RunAnalysis() error = %v, want *domain.InternalError", err)
	}
}

func TestCollectErrorPatterns_NoPatterns_ReturnsEmptySlice(t *testing.T) {
	got := collectErrorPatterns([]analysis.Fragment{{SourceES: "x"}})
	if got == nil {
		t.Fatal("collectErrorPatterns() = nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("collectErrorPatterns() = %v, want empty", got)
	}
}
