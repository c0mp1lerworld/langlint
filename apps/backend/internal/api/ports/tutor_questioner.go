package ports

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// QuestionRequest is the anonymized context used to generate an active-practice
// question anchored to a fragment's error (PRODUCT_DOMAIN §1.2).
type QuestionRequest struct {
	SourceText    string
	UserDraft     string
	Correction    string
	ErrorPatterns []domain.ErrorPattern
	TargetRules   []practice.TargetRule
}

// EvaluateRequest is the context used to grade a learner's answer to a
// previously generated question.
type EvaluateRequest struct {
	SourceText    string
	UserDraft     string
	Correction    string
	ErrorPatterns []domain.ErrorPattern
	Question      string
	Answer        string
}

// TutorQuestioner is the port of the active-practice engine: it generates a
// question for a fragment's error and evaluates the learner's answer. Like the
// LLMExtractor, it is provider-agnostic (A1) and runs outside any transaction.
type TutorQuestioner interface {
	Question(ctx context.Context, req QuestionRequest) (analysis.QuizQuestion, error)
	Evaluate(ctx context.Context, req EvaluateRequest) (analysis.QuizEvaluation, error)
}
