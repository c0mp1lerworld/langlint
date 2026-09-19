package event_handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

type fakeRunner struct {
	called bool
	id     domain.ID
	err    error
}

func (f *fakeRunner) RunAnalysis(_ context.Context, id domain.ID) error {
	f.called = true
	f.id = id
	return f.err
}

func analysisPractice(userID domain.ID, status practice.PracticeStatus) *practice.Practice {
	return &practice.Practice{
		ID:          domain.MustNewID(),
		UserID:      userID,
		SourceText:  practice.SourceText("El perro escapó."),
		DraftText:   practice.DraftText("The dog escaped."),
		TargetRules: []practice.TargetRule{{Verb: "run"}},
		Status:      status,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
}

func TestAnalysisRequestedHandler_Analyzing_RunsAnalysis(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(nil, &domain.NotFoundError{Field: "analysis"})

	runner := &fakeRunner{}
	h := NewAnalysisRequestedHandler(practices, analyses, runner)
	if err := h.Handle(context.Background(), domain.AnalysisRequested{PracticeID: p.ID, UserID: userID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !runner.called || runner.id != p.ID {
		t.Fatalf("runner.called = %v, id = %v", runner.called, runner.id)
	}
}

func TestAnalysisRequestedHandler_NotAnalyzing_Skips(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	runner := &fakeRunner{}
	h := NewAnalysisRequestedHandler(practices, mocks.NewMockAnalysisRepository(ctrl), runner)
	if err := h.Handle(context.Background(), domain.AnalysisRequested{PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if runner.called {
		t.Fatal("runner was called for a non-analyzing practice")
	}
}

func TestAnalysisRequestedHandler_AnalysisExists_Skips(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(&analysis.Analysis{ID: domain.MustNewID(), PracticeID: p.ID}, nil)

	runner := &fakeRunner{}
	h := NewAnalysisRequestedHandler(practices, analyses, runner)
	if err := h.Handle(context.Background(), domain.AnalysisRequested{PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if runner.called {
		t.Fatal("runner was called although an analysis already exists")
	}
}

func TestAnalysisRequestedHandler_PracticeGone_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.NotFoundError{Field: "practice"})

	h := NewAnalysisRequestedHandler(practices, mocks.NewMockAnalysisRepository(ctrl), &fakeRunner{})
	if err := h.Handle(context.Background(), domain.AnalysisRequested{PracticeID: id}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}

func TestAnalysisRequestedHandler_WrongEvent_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAnalysisRequestedHandler(mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), &fakeRunner{})
	if err := h.Handle(context.Background(), domain.PracticeCreated{}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}

func TestAnalysisCompletedHandler_Analyzing_MarksCompletedAndUpsertsMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *practice.Practice) error {
		if got.Status != practice.PracticeStatusCompleted {
			t.Fatalf("saved status = %q, want completed", got.Status)
		}
		return nil
	})

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, gomock.Any()).Return(nil, nil).Times(3)
	metrics.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m *analytics.ErrorMetric) error {
		if m.Window != analytics.WindowDay && m.Window != analytics.WindowWeek && m.Window != analytics.WindowMonth {
			t.Fatalf("upsert window = %q", m.Window)
		}
		return nil
	}).Times(3)

	h := NewAnalysisCompletedHandler(practices, metrics)
	event := domain.AnalysisCompleted{
		UserID:        userID,
		PracticeID:    p.ID,
		ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeTenseAgreement, Severity: domain.ErrorPatternSeverityMinor}},
	}
	if err := h.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestAnalysisCompletedHandler_NoPatterns_SkipsMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	// metrics repository is never touched: no ListByUser/Upsert expectations.
	h := NewAnalysisCompletedHandler(practices, mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), domain.AnalysisCompleted{UserID: userID, PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestAnalysisCompletedHandler_NotAnalyzing_Skips(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusCompleted)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	h := NewAnalysisCompletedHandler(practices, mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), domain.AnalysisCompleted{PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}

func TestAnalysisCompletedHandler_MetricsFail_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	metrics.EXPECT().ListByUser(gomock.Any(), userID, gomock.Any()).Return(nil, errors.New("db down"))

	h := NewAnalysisCompletedHandler(practices, metrics)
	event := domain.AnalysisCompleted{UserID: userID, PracticeID: p.ID, ErrorPatterns: []domain.ErrorPattern{{Code: domain.ErrorPatternCodeWordOrder}}}
	if err := h.Handle(context.Background(), event); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisFailedHandler_Analyzing_MarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *practice.Practice) error {
		if got.Status != practice.PracticeStatusFailed {
			t.Fatalf("saved status = %q, want failed", got.Status)
		}
		return nil
	})

	h := NewAnalysisFailedHandler(practices)
	if err := h.Handle(context.Background(), domain.AnalysisFailed{PracticeID: p.ID, Reason: "llm unavailable"}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestAnalysisFailedHandler_NotAnalyzing_Skips(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusCompleted)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	h := NewAnalysisFailedHandler(practices)
	if err := h.Handle(context.Background(), domain.AnalysisFailed{PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}

func TestAnalysisRequestedHandler_PointerEvent_Runs(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(nil, &domain.NotFoundError{Field: "analysis"})

	runner := &fakeRunner{}
	h := NewAnalysisRequestedHandler(practices, analyses, runner)
	if err := h.Handle(context.Background(), &domain.AnalysisRequested{PracticeID: p.ID, UserID: userID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !runner.called {
		t.Fatal("runner was not called for a pointer event")
	}
}

func TestAnalysisRequestedHandler_PracticeError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.InternalError{Field: "practice"})

	h := NewAnalysisRequestedHandler(practices, mocks.NewMockAnalysisRepository(ctrl), &fakeRunner{})
	if err := h.Handle(context.Background(), domain.AnalysisRequested{PracticeID: id}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisRequestedHandler_AnalysisLookupError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(nil, &domain.InternalError{Field: "analysis"})

	h := NewAnalysisRequestedHandler(practices, analyses, &fakeRunner{})
	if err := h.Handle(context.Background(), domain.AnalysisRequested{PracticeID: p.ID}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisCompletedHandler_PointerEvent_MarksCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	h := NewAnalysisCompletedHandler(practices, mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), &domain.AnalysisCompleted{UserID: userID, PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestAnalysisCompletedHandler_PracticeError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.InternalError{Field: "practice"})

	h := NewAnalysisCompletedHandler(practices, mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), domain.AnalysisCompleted{PracticeID: id}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisCompletedHandler_SaveFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	h := NewAnalysisCompletedHandler(practices, mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), domain.AnalysisCompleted{PracticeID: p.ID}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisCompletedHandler_PracticeGone_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.NotFoundError{Field: "practice"})

	h := NewAnalysisCompletedHandler(practices, mocks.NewMockErrorMetricRepository(ctrl))
	if err := h.Handle(context.Background(), domain.AnalysisCompleted{PracticeID: id}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}

func TestAnalysisFailedHandler_PointerEvent_MarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	h := NewAnalysisFailedHandler(practices)
	if err := h.Handle(context.Background(), &domain.AnalysisFailed{PracticeID: p.ID}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestAnalysisFailedHandler_PracticeError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.InternalError{Field: "practice"})

	h := NewAnalysisFailedHandler(practices)
	if err := h.Handle(context.Background(), domain.AnalysisFailed{PracticeID: id}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisFailedHandler_SaveFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := analysisPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	h := NewAnalysisFailedHandler(practices)
	if err := h.Handle(context.Background(), domain.AnalysisFailed{PracticeID: p.ID}); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestAnalysisFailedHandler_WrongEvent_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAnalysisFailedHandler(mocks.NewMockPracticeRepository(ctrl))
	if err := h.Handle(context.Background(), domain.PracticeCreated{}); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}
