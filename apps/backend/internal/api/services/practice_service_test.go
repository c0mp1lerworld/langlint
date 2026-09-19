package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// runUoW returns a UnitOfWork mock that executes the closure in place.
func runUoW(ctrl *gomock.Controller) *mocks.MockUnitOfWork {
	uow := mocks.NewMockUnitOfWork(ctrl)
	uow.EXPECT().InTransaction(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	).AnyTimes()
	return uow
}

func ownedPractice(userID domain.ID, status practice.PracticeStatus) *practice.Practice {
	return &practice.Practice{
		ID:          domain.MustNewID(),
		UserID:      userID,
		SourceText:  practice.SourceText("El perro escapó."),
		DraftText:   practice.DraftText("The dog escaped."),
		TargetRules: []practice.TargetRule{{Verb: "run", Tense: "past simple"}},
		Status:      status,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
}

func validInput() PracticeInput {
	return PracticeInput{
		SourceText:  "El perro escapó.",
		DraftText:   "The dog escaped.",
		TargetRules: []TargetRuleInput{{Verb: "run", Tense: "past simple"}},
	}
}

func TestPracticeService_Create_Valid_SavesAndAppendsPracticeCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()

	practices := mocks.NewMockPracticeRepository(ctrl)
	var saved *practice.Practice
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *practice.Practice) error {
			saved = p
			return nil
		},
	)

	outbox := mocks.NewMockOutbox(ctrl)
	var appended domain.DomainEvent
	outbox.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e domain.DomainEvent) error {
			appended = e
			return nil
		},
	)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), outbox)
	got, err := svc.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if saved == nil || saved.ID != got.ID {
		t.Fatalf("saved practice = %v, want id %v", saved, got.ID)
	}
	if got.Status != practice.PracticeStatusDraft {
		t.Fatalf("status = %q, want draft", got.Status)
	}
	if got.UserID != userID {
		t.Fatalf("user id = %v, want %v", got.UserID, userID)
	}

	event, ok := appended.(domain.PracticeCreated)
	if !ok {
		t.Fatalf("appended event = %T, want domain.PracticeCreated", appended)
	}
	if event.PracticeID != got.ID || event.UserID != userID {
		t.Fatalf("event = %+v, want practice %v user %v", event, got.ID, userID)
	}
}

func TestPracticeService_Create_EmptySource_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	in := validInput()
	in.SourceText = ""

	svc := NewPracticeService(
		runUoW(ctrl),
		mocks.NewMockPracticeRepository(ctrl),
		mocks.NewMockAnalysisRepository(ctrl),
		mocks.NewMockOutbox(ctrl),
	)
	if _, err := svc.Create(context.Background(), domain.MustNewID(), in); !isValidation(err) {
		t.Fatalf("Create(empty source) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Create_EmptyRules_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	in := validInput()
	in.TargetRules = nil

	svc := NewPracticeService(
		runUoW(ctrl),
		mocks.NewMockPracticeRepository(ctrl),
		mocks.NewMockAnalysisRepository(ctrl),
		mocks.NewMockOutbox(ctrl),
	)
	if _, err := svc.Create(context.Background(), domain.MustNewID(), in); !isValidation(err) {
		t.Fatalf("Create(no rules) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Get_WithAnalysis_ReturnsBoth(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusCompleted)
	a := &analysis.Analysis{ID: domain.MustNewID(), PracticeID: p.ID}

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(a, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, analyses, mocks.NewMockOutbox(ctrl))
	gotP, gotA, err := svc.Get(context.Background(), userID, p.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotP != p || gotA != a {
		t.Fatalf("Get() = (%v, %v), want (%v, %v)", gotP, gotA, p, a)
	}
}

func TestPracticeService_Get_NoAnalysis_ReturnsNilAnalysis(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(nil, &domain.NotFoundError{Field: "analysis"})

	svc := NewPracticeService(runUoW(ctrl), practices, analyses, mocks.NewMockOutbox(ctrl))
	gotP, gotA, err := svc.Get(context.Background(), userID, p.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotP != p {
		t.Fatalf("practice = %v, want %v", gotP, p)
	}
	if gotA != nil {
		t.Fatalf("analysis = %v, want nil", gotA)
	}
}

func TestPracticeService_Get_OtherUser_ReturnsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	owner := domain.MustNewID()
	other := domain.MustNewID()
	p := ownedPractice(owner, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, _, err := svc.Get(context.Background(), other, p.ID); !isNotFound(err) {
		t.Fatalf("Get(other user) error = %v, want NotFoundError", err)
	}
}

func TestPracticeService_List_DelegatesToRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	want := []practice.Practice{*ownedPractice(userID, practice.PracticeStatusDraft)}

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().ListByUser(gomock.Any(), userID, 20, 0).Return(want, 1, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	got, total, err := svc.List(context.Background(), userID, 20, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != want[0].ID {
		t.Fatalf("List() = (%v, %d), want (%v, 1)", got, total, want)
	}
}

func TestPracticeService_Update_PartialEdit_Saves(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	var saved *practice.Practice
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *practice.Practice) error {
			saved = got
			return nil
		},
	)

	newDraft := "The dog has escaped."
	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{DraftText: &newDraft}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if saved.DraftText.String() != newDraft {
		t.Fatalf("draft = %q, want %q", saved.DraftText, newDraft)
	}
	if saved.SourceText != p.SourceText {
		t.Fatalf("source changed unexpectedly: %q", saved.SourceText)
	}
}

func TestPracticeService_Update_NotDraft_ReturnsInvalidState(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusCompleted)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	newDraft := "x"
	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{DraftText: &newDraft}); !isInvalidState(err) {
		t.Fatalf("Update(completed) error = %v, want InvalidStateError", err)
	}
}

func TestPracticeService_Delete_Draft_SoftDeletes(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), p).Return(nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Delete(context.Background(), userID, p.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if p.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want soft-delete timestamp")
	}
}

func TestPracticeService_Delete_Analyzing_ReturnsInvalidState(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Delete(context.Background(), userID, p.ID); !isInvalidState(err) {
		t.Fatalf("Delete(analyzing) error = %v, want InvalidStateError", err)
	}
}

func TestPracticeService_Analyze_Draft_StartsAndAppendsAnalysisRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *practice.Practice) error {
			if got.Status != practice.PracticeStatusAnalyzing {
				t.Fatalf("saved status = %q, want analyzing", got.Status)
			}
			return nil
		},
	)

	outbox := mocks.NewMockOutbox(ctrl)
	var appended domain.DomainEvent
	outbox.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e domain.DomainEvent) error {
			appended = e
			return nil
		},
	)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), outbox)
	if err := svc.Analyze(context.Background(), userID, p.ID); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	event, ok := appended.(domain.AnalysisRequested)
	if !ok {
		t.Fatalf("appended event = %T, want domain.AnalysisRequested", appended)
	}
	if event.PracticeID != p.ID {
		t.Fatalf("event practice = %v, want %v", event.PracticeID, p.ID)
	}
}

func TestPracticeService_Analyze_AlreadyAnalyzing_ReturnsPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusAnalyzing)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Analyze(context.Background(), userID, p.ID); !isPending(err) {
		t.Fatalf("Analyze(analyzing) error = %v, want AnalysisPendingError", err)
	}
}

func TestPracticeService_Analyze_Completed_ReturnsInvalidState(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusCompleted)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Analyze(context.Background(), userID, p.ID); !isInvalidState(err) {
		t.Fatalf("Analyze(completed) error = %v, want InvalidStateError", err)
	}
}

func isValidation(err error) bool {
	var target *domain.ValidationError
	return errors.As(err, &target)
}

func isNotFound(err error) bool {
	var target *domain.NotFoundError
	return errors.As(err, &target)
}

func isInvalidState(err error) bool {
	var target *domain.InvalidStateError
	return errors.As(err, &target)
}

func isPending(err error) bool {
	var target *domain.AnalysisPendingError
	return errors.As(err, &target)
}
