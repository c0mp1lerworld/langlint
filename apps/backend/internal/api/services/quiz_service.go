package services

import (
	"context"
	"strings"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// QuizService exposes the active-practice loop (PRODUCT_DOMAIN §1.2): it
// generates a question anchored to a fragment's error and evaluates the
// learner's answer. Each evaluation is recorded as an append-only quiz attempt
// so the learner's hits and misses feed the analytics (checklist 9.7). It
// depends on ports only (A2).
type QuizService struct {
	practices  storage.PracticeRepository
	analyses   storage.AnalysisRepository
	questioner ports.TutorQuestioner
	attempts   storage.QuizAttemptRepository
	now        func() time.Time
}

// NewQuizService wires the active-practice use cases through their ports.
func NewQuizService(practices storage.PracticeRepository, analyses storage.AnalysisRepository, questioner ports.TutorQuestioner, attempts storage.QuizAttemptRepository) *QuizService {
	return &QuizService{
		practices:  practices,
		analyses:   analyses,
		questioner: questioner,
		attempts:   attempts,
		now:        time.Now,
	}
}

// Question generates one question for the fragment at index.
func (s *QuizService) Question(ctx context.Context, practiceID domain.ID, index int) (analysis.QuizQuestion, error) {
	practice, fragment, err := s.load(ctx, practiceID, index)
	if err != nil {
		return analysis.QuizQuestion{}, err
	}
	return s.questioner.Question(ctx, ports.QuestionRequest{
		SourceText:    fragment.SourceES,
		UserDraft:     fragment.UserDraft,
		Correction:    fragment.Correction,
		ErrorPatterns: fragment.ErrorPatterns,
		TargetRules:   practice.TargetRules,
	})
}

// Evaluate grades the learner's answer to a previously generated question and
// records the result as a quiz attempt (9.7). The evaluation is stateless: the
// client sends back the question it is answering. The recorded attempt is
// append-only; a failure to record it is propagated so no attempt is silently
// lost.
func (s *QuizService) Evaluate(ctx context.Context, practiceID domain.ID, index int, question, answer string) (analysis.QuizEvaluation, error) {
	if strings.TrimSpace(question) == "" || strings.TrimSpace(answer) == "" {
		return analysis.QuizEvaluation{}, &domain.ValidationError{Field: "answer", Message: "question and answer must not be empty"}
	}

	p, fragment, err := s.load(ctx, practiceID, index)
	if err != nil {
		return analysis.QuizEvaluation{}, err
	}

	evaluation, err := s.questioner.Evaluate(ctx, ports.EvaluateRequest{
		SourceText:    fragment.SourceES,
		UserDraft:     fragment.UserDraft,
		Correction:    fragment.Correction,
		ErrorPatterns: fragment.ErrorPatterns,
		Question:      question,
		Answer:        answer,
	})
	if err != nil {
		return analysis.QuizEvaluation{}, err
	}

	attempt, err := analytics.NewQuizAttempt(p.UserID, practiceID, evaluation.Correct, s.now())
	if err != nil {
		return analysis.QuizEvaluation{}, err
	}
	if err := s.attempts.Append(ctx, attempt); err != nil {
		return analysis.QuizEvaluation{}, err
	}
	return evaluation, nil
}

// Stats returns the learner's aggregated quiz performance (9.7): total
// attempts, correct attempts and accuracy derived on read from the append-only
// ledger.
func (s *QuizService) Stats(ctx context.Context, userID domain.ID) (analytics.QuizStats, error) {
	attempts, err := s.attempts.ListByUser(ctx, userID)
	if err != nil {
		return analytics.QuizStats{}, err
	}
	return analytics.BuildQuizStats(attempts), nil
}

// load fetches the practice, its analysis and the fragment at index. The
// analysis must exist (the practice must have been analyzed first).
func (s *QuizService) load(ctx context.Context, practiceID domain.ID, index int) (*practice.Practice, analysis.Fragment, error) {
	p, err := s.practices.GetByID(ctx, practiceID)
	if err != nil {
		return nil, analysis.Fragment{}, err
	}

	a, err := s.analyses.GetByPracticeID(ctx, practiceID)
	if err != nil {
		return nil, analysis.Fragment{}, err
	}
	if index < 0 || index >= len(a.Fragments) {
		return nil, analysis.Fragment{}, &domain.ValidationError{Field: "fragment_index", Message: "is out of range"}
	}
	return p, a.Fragments[index], nil
}
