package analysis_test

import (
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

func TestQuizKind_IsValid(t *testing.T) {
	for _, kind := range []analysis.QuizKind{analysis.QuizKindOpen, analysis.QuizKindFill} {
		if !kind.IsValid() {
			t.Fatalf("QuizKind(%q).IsValid() = false, want true", kind)
		}
	}
	if analysis.QuizKind("mcq").IsValid() {
		t.Fatal("QuizKind(mcq).IsValid() = true, want false")
	}
}

func TestNewQuizQuestion_Valid_TrimsPrompt(t *testing.T) {
	question, err := analysis.NewQuizQuestion(analysis.QuizKindFill, "  Completa: I bet ___ my team.  ")
	if err != nil {
		t.Fatalf("NewQuizQuestion() error = %v", err)
	}
	if question.Kind != analysis.QuizKindFill {
		t.Fatalf("Kind = %q, want fill", question.Kind)
	}
	if question.Prompt != "Completa: I bet ___ my team." {
		t.Fatalf("Prompt = %q, want trimmed", question.Prompt)
	}
}

func TestNewQuizQuestion_InvalidKind_ReturnsValidationError(t *testing.T) {
	_, err := analysis.NewQuizQuestion(analysis.QuizKind("mcq"), "x")
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.ValidationError", err)
	}
}

func TestNewQuizQuestion_EmptyPrompt_ReturnsValidationError(t *testing.T) {
	_, err := analysis.NewQuizQuestion(analysis.QuizKindOpen, "   ")
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.ValidationError", err)
	}
}

func TestNewQuizEvaluation_Valid_TrimsFollowUp(t *testing.T) {
	evaluation, err := analysis.NewQuizEvaluation(true, "  Muy bien.  ", "  ¿Por qué?  ")
	if err != nil {
		t.Fatalf("NewQuizEvaluation() error = %v", err)
	}
	if !evaluation.Correct {
		t.Fatal("Correct = false, want true")
	}
	if evaluation.Feedback != "Muy bien." || evaluation.FollowUp != "¿Por qué?" {
		t.Fatalf("evaluation = %+v, want trimmed fields", evaluation)
	}
}

func TestNewQuizEvaluation_EmptyFollowUp_IsAllowed(t *testing.T) {
	evaluation, err := analysis.NewQuizEvaluation(false, "Casi.", "")
	if err != nil {
		t.Fatalf("NewQuizEvaluation() error = %v", err)
	}
	if evaluation.FollowUp != "" {
		t.Fatalf("FollowUp = %q, want empty", evaluation.FollowUp)
	}
}

func TestNewQuizEvaluation_EmptyFeedback_ReturnsValidationError(t *testing.T) {
	_, err := analysis.NewQuizEvaluation(true, "  ", "")
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.ValidationError", err)
	}
}
