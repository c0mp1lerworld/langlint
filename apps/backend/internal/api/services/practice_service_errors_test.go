package services

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func strPtr(v string) *string { return &v }

func TestPracticeService_Create_SaveFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Create(context.Background(), domain.MustNewID(), validInput()); err == nil {
		t.Fatal("Create() error = nil, want error")
	}
}

func TestPracticeService_Create_TransactionFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	uow := mocks.NewMockUnitOfWork(ctrl)
	uow.EXPECT().InTransaction(gomock.Any(), gomock.Any()).Return(errors.New("tx failed"))

	svc := NewPracticeService(uow, mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Create(context.Background(), domain.MustNewID(), validInput()); err == nil {
		t.Fatal("Create() error = nil, want error")
	}
}

func TestPracticeService_Create_EmptyDraft_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	in := validInput()
	in.DraftText = ""

	svc := NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Create(context.Background(), domain.MustNewID(), in); !isValidation(err) {
		t.Fatalf("Create(empty draft) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Create_InvalidTargetRule_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	in := validInput()
	in.TargetRules = []TargetRuleInput{{Verb: ""}}

	svc := NewPracticeService(runUoW(ctrl), mocks.NewMockPracticeRepository(ctrl), mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Create(context.Background(), domain.MustNewID(), in); !isValidation(err) {
		t.Fatalf("Create(invalid rule) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Get_RepositoryError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.InternalError{Field: "practice"})

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, _, err := svc.Get(context.Background(), domain.MustNewID(), id); err == nil {
		t.Fatal("Get() error = nil, want error")
	}
}

func TestPracticeService_Get_AnalysisError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(nil, &domain.InternalError{Field: "analysis"})

	svc := NewPracticeService(runUoW(ctrl), practices, analyses, mocks.NewMockOutbox(ctrl))
	if _, _, err := svc.Get(context.Background(), userID, p.ID); err == nil {
		t.Fatal("Get() error = nil, want error")
	}
}

func TestPracticeService_Update_InvalidSource_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{SourceText: strPtr("")}); !isValidation(err) {
		t.Fatalf("Update(empty source) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Update_InvalidDraft_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{DraftText: strPtr("")}); !isValidation(err) {
		t.Fatalf("Update(empty draft) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Update_EmptyRules_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	empty := []TargetRuleInput{}
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{TargetRules: &empty}); !isValidation(err) {
		t.Fatalf("Update(empty rules) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Update_InvalidTargetRule_ReturnsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	bad := []TargetRuleInput{{Verb: ""}}
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{TargetRules: &bad}); !isValidation(err) {
		t.Fatalf("Update(invalid rule) error = %v, want ValidationError", err)
	}
}

func TestPracticeService_Update_RepositoryError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.InternalError{Field: "practice"})

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Update(context.Background(), domain.MustNewID(), id, PracticeUpdateInput{DraftText: strPtr("x")}); err == nil {
		t.Fatal("Update() error = nil, want error")
	}
}

func TestPracticeService_Update_SaveFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if _, err := svc.Update(context.Background(), userID, p.ID, PracticeUpdateInput{DraftText: strPtr("x")}); err == nil {
		t.Fatal("Update() error = nil, want error")
	}
}

func TestPracticeService_Delete_RepositoryError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.NotFoundError{Field: "practice"})

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Delete(context.Background(), domain.MustNewID(), id); err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
}

func TestPracticeService_Delete_SaveFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)
	practices.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Delete(context.Background(), userID, p.ID); err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
}

func TestPracticeService_Analyze_RepositoryError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	id := domain.MustNewID()
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), id).Return(nil, &domain.NotFoundError{Field: "practice"})

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Analyze(context.Background(), domain.MustNewID(), id); err == nil {
		t.Fatal("Analyze() error = nil, want error")
	}
}

func TestPracticeService_Analyze_TransactionFails_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	userID := domain.MustNewID()
	p := ownedPractice(userID, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	uow := mocks.NewMockUnitOfWork(ctrl)
	uow.EXPECT().InTransaction(gomock.Any(), gomock.Any()).Return(errors.New("tx failed"))

	svc := NewPracticeService(uow, practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Analyze(context.Background(), userID, p.ID); err == nil {
		t.Fatal("Analyze() error = nil, want error")
	}
}

func TestPracticeService_Analyze_OtherUser_ReturnsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	owner := domain.MustNewID()
	p := ownedPractice(owner, practice.PracticeStatusDraft)

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil)

	svc := NewPracticeService(runUoW(ctrl), practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockOutbox(ctrl))
	if err := svc.Analyze(context.Background(), domain.MustNewID(), p.ID); !isNotFound(err) {
		t.Fatalf("Analyze(other user) error = %v, want NotFoundError", err)
	}
}
