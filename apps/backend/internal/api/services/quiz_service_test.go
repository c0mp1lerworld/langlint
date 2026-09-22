package services

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func quizPractice() *practice.Practice {
	return &practice.Practice{
		ID:          domain.MustNewID(),
		UserID:      domain.MustNewID(),
		SourceText:  practice.SourceText("Ayer aposté en las carreras."),
		DraftText:   practice.DraftText("Yesterday I bet in the races."),
		TargetRules: []practice.TargetRule{{Verb: "bet"}},
		Status:      practice.PracticeStatusCompleted,
	}
}

func quizAnalysis(practiceID domain.ID) *analysis.Analysis {
	return &analysis.Analysis{
		ID:         domain.MustNewID(),
		PracticeID: practiceID,
		Status:     analysis.AnalysisStatusCompleted,
		Fragments: []analysis.Fragment{{
			SourceES:   "Ayer aposté en las carreras.",
			UserDraft:  "Yesterday I bet in the races.",
			Correction: "Yesterday I bet on the races.",
			ErrorPatterns: []domain.ErrorPattern{{
				Code:     domain.ErrorPatternCodePrepositionInfinitive,
				Severity: domain.ErrorPatternSeverityModerate,
			}},
		}},
	}
}

type quizFixture struct {
	svc        *QuizService
	practices  *mocks.MockPracticeRepository
	analyses   *mocks.MockAnalysisRepository
	questioner *mocks.MockTutorQuestioner
	attempts   *mocks.MockQuizAttemptRepository
	practice   *practice.Practice
}

func newQuizFixture(t *testing.T) *quizFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	p := quizPractice()
	analyses := mocks.NewMockAnalysisRepository(ctrl)
	analyses.EXPECT().GetByPracticeID(gomock.Any(), p.ID).Return(quizAnalysis(p.ID), nil).AnyTimes()

	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), p.ID).Return(p, nil).AnyTimes()

	questioner := mocks.NewMockTutorQuestioner(ctrl)
	attempts := mocks.NewMockQuizAttemptRepository(ctrl)
	return &quizFixture{
		svc:        NewQuizService(practices, analyses, questioner, attempts),
		practices:  practices,
		analyses:   analyses,
		questioner: questioner,
		attempts:   attempts,
		practice:   p,
	}
}

func TestQuizService_Question_ReturnsGeneratedQuestion(t *testing.T) {
	f := newQuizFixture(t)
	want := analysis.QuizQuestion{Kind: analysis.QuizKindFill, Prompt: "I bet ___ the races."}

	f.questioner.EXPECT().Question(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req ports.QuestionRequest) (analysis.QuizQuestion, error) {
			if req.UserDraft != "Yesterday I bet in the races." {
				t.Fatalf("UserDraft = %q", req.UserDraft)
			}
			if len(req.ErrorPatterns) != 1 || req.ErrorPatterns[0].Code != domain.ErrorPatternCodePrepositionInfinitive {
				t.Fatalf("ErrorPatterns = %+v", req.ErrorPatterns)
			}
			if len(req.TargetRules) != 1 || req.TargetRules[0].Verb != "bet" {
				t.Fatalf("TargetRules = %+v", req.TargetRules)
			}
			return want, nil
		},
	)

	got, err := f.svc.Question(context.Background(), f.practice.ID, 0)
	if err != nil {
		t.Fatalf("Question() error = %v", err)
	}
	if got != want {
		t.Fatalf("Question() = %+v, want %+v", got, want)
	}
}

func TestQuizService_Question_FragmentIndexOutOfRange_ReturnsValidationError(t *testing.T) {
	f := newQuizFixture(t)

	_, err := f.svc.Question(context.Background(), f.practice.ID, 3)
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.ValidationError", err)
	}
}

func TestQuizService_Question_PracticeRepositoryError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	practices := mocks.NewMockPracticeRepository(ctrl)
	practices.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, &domain.NotFoundError{Field: "practice"})
	svc := NewQuizService(practices, mocks.NewMockAnalysisRepository(ctrl), mocks.NewMockTutorQuestioner(ctrl), mocks.NewMockQuizAttemptRepository(ctrl))

	_, err := svc.Question(context.Background(), domain.MustNewID(), 0)
	var target *domain.NotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.NotFoundError", err)
	}
}

func TestQuizService_Evaluate_ReturnsEvaluation(t *testing.T) {
	f := newQuizFixture(t)
	want := analysis.QuizEvaluation{Correct: true, Feedback: "¡Correcto!", FollowUp: ""}

	f.questioner.EXPECT().Evaluate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req ports.EvaluateRequest) (analysis.QuizEvaluation, error) {
			if req.Question != "I bet ___ the races." || req.Answer != "on" {
				t.Fatalf("req = %+v", req)
			}
			return want, nil
		},
	)
	f.attempts.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	got, err := f.svc.Evaluate(context.Background(), f.practice.ID, 0, "I bet ___ the races.", "on")
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got != want {
		t.Fatalf("Evaluate() = %+v, want %+v", got, want)
	}
}

func TestQuizService_Evaluate_RecordsAttempt(t *testing.T) {
	f := newQuizFixture(t)

	f.questioner.EXPECT().Evaluate(gomock.Any(), gomock.Any()).Return(
		analysis.QuizEvaluation{Correct: true, Feedback: "¡Correcto!"}, nil)
	f.attempts.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, attempt *analytics.QuizAttempt) error {
			if attempt.UserID != f.practice.UserID {
				t.Fatalf("attempt.UserID = %s, want %s", attempt.UserID, f.practice.UserID)
			}
			if attempt.PracticeID != f.practice.ID {
				t.Fatalf("attempt.PracticeID = %s, want %s", attempt.PracticeID, f.practice.ID)
			}
			if !attempt.Correct {
				t.Fatal("attempt.Correct = false, want true")
			}
			return nil
		},
	)

	if _, err := f.svc.Evaluate(context.Background(), f.practice.ID, 0, "q", "a"); err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
}

func TestQuizService_Evaluate_AppendError_Propagates(t *testing.T) {
	f := newQuizFixture(t)

	f.questioner.EXPECT().Evaluate(gomock.Any(), gomock.Any()).Return(
		analysis.QuizEvaluation{Correct: false, Feedback: "Casi."}, nil)
	f.attempts.EXPECT().Append(gomock.Any(), gomock.Any()).Return(&domain.InternalError{Field: "quiz_attempt"})

	if _, err := f.svc.Evaluate(context.Background(), f.practice.ID, 0, "q", "a"); err == nil {
		t.Fatal("Evaluate() error = nil, want error")
	}
}

func TestQuizService_Stats_ReturnsAggregatedStats(t *testing.T) {
	f := newQuizFixture(t)

	f.attempts.EXPECT().ListByUser(gomock.Any(), f.practice.UserID).Return(
		[]analytics.QuizAttempt{{Correct: true}, {Correct: false}}, nil)

	stats, err := f.svc.Stats(context.Background(), f.practice.UserID)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.TotalAttempts != 2 || stats.CorrectAttempts != 1 {
		t.Fatalf("stats = %+v, want total=2 correct=1", stats)
	}
}

func TestQuizService_Evaluate_EmptyAnswer_ReturnsValidationError(t *testing.T) {
	f := newQuizFixture(t)

	_, err := f.svc.Evaluate(context.Background(), f.practice.ID, 0, "I bet ___ the races.", "   ")
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.ValidationError", err)
	}
}
