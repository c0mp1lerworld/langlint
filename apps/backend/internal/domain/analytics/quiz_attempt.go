package analytics

import (
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// generateID is a seam for deterministic tests; it defaults to domain.NewID.
var generateID = domain.NewID

// QuizAttempt is an append-only record of a quiz answer (checklist 9.7): it
// captures whether the learner got the question right, anchored to the practice
// it belonged to. Like the other analytics ledgers it is never updated nor
// deleted (A4); the right to be forgotten removes it wholesale (A9). It is a
// separate source of truth from error_metrics, so it never needs to enter the
// refresh-aggregates reconciliation (which rebuilds error_metrics from analyses
// only).
type QuizAttempt struct {
	ID         domain.ID `json:"id"`
	UserID     domain.ID `json:"user_id"`
	PracticeID domain.ID `json:"practice_id"`
	Correct    bool      `json:"correct"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewQuizAttempt records a quiz answer, generating its UUID v7. userID and
// practiceID must be set and createdAt must not be zero.
func NewQuizAttempt(userID, practiceID domain.ID, correct bool, createdAt time.Time) (*QuizAttempt, error) {
	if userID.IsZero() {
		return nil, &domain.ValidationError{Field: "user_id", Message: "must not be empty"}
	}
	if practiceID.IsZero() {
		return nil, &domain.ValidationError{Field: "practice_id", Message: "must not be empty"}
	}
	if createdAt.IsZero() {
		return nil, &domain.ValidationError{Field: "created_at", Message: "must not be zero"}
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &QuizAttempt{
		ID:         id,
		UserID:     userID,
		PracticeID: practiceID,
		Correct:    correct,
		CreatedAt:  createdAt.UTC(),
	}, nil
}

// QuizStats is the read model of the learner's quiz performance (checklist 9.7):
// how many answers were given, how many were correct, and the accuracy derived
// from those counts.
type QuizStats struct {
	TotalAttempts   int     `json:"total_attempts"`
	CorrectAttempts int     `json:"correct_attempts"`
	Accuracy        float64 `json:"accuracy"`
}

// BuildQuizStats aggregates the attempts into the learner's quiz stats. With no
// attempts there is nothing to get right or wrong, so accuracy is 1 (matching
// the progress series convention).
func BuildQuizStats(attempts []QuizAttempt) QuizStats {
	total := len(attempts)
	correct := 0
	for _, attempt := range attempts {
		if attempt.Correct {
			correct++
		}
	}
	return QuizStats{
		TotalAttempts:   total,
		CorrectAttempts: correct,
		Accuracy:        accuracy(total, total-correct),
	}
}
