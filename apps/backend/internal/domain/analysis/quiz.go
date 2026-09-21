package analysis

import (
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// QuizKind is the type of active-practice question the tutor generates
// (PRODUCT_DOMAIN §1.2). Only production-oriented kinds are supported; a
// recognition-only multiple choice is intentionally out of scope.
type QuizKind string

// Known quiz kinds.
const (
	QuizKindOpen QuizKind = "open"
	QuizKindFill QuizKind = "fill"
)

// IsValid reports whether the kind is one of the known values.
func (k QuizKind) IsValid() bool {
	switch k {
	case QuizKindOpen, QuizKindFill:
		return true
	default:
		return false
	}
}

// QuizQuestion is an active-practice question anchored to a fragment.
type QuizQuestion struct {
	Kind   QuizKind `json:"kind"`
	Prompt string   `json:"prompt"`
}

// NewQuizQuestion validates a generated question and returns it.
func NewQuizQuestion(kind QuizKind, prompt string) (QuizQuestion, error) {
	if !kind.IsValid() {
		return QuizQuestion{}, &domain.ValidationError{Field: "quiz.kind", Message: "must be open or fill"}
	}
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return QuizQuestion{}, &domain.ValidationError{Field: "quiz.prompt", Message: "must not be empty"}
	}
	return QuizQuestion{Kind: kind, Prompt: trimmed}, nil
}

// QuizEvaluation is the result of evaluating a learner's answer.
type QuizEvaluation struct {
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback"`
	FollowUp string `json:"follow_up"`
}

// NewQuizEvaluation validates an evaluation and returns it. FollowUp may be
// empty (no follow-up question); feedback must carry content.
func NewQuizEvaluation(correct bool, feedback, followUp string) (QuizEvaluation, error) {
	trimmed := strings.TrimSpace(feedback)
	if trimmed == "" {
		return QuizEvaluation{}, &domain.ValidationError{Field: "quiz.feedback", Message: "must not be empty"}
	}
	return QuizEvaluation{
		Correct:  correct,
		Feedback: trimmed,
		FollowUp: strings.TrimSpace(followUp),
	}, nil
}
