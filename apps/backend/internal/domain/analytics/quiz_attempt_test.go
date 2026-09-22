package analytics

import (
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestNewQuizAttempt_Valid_ReturnsAttempt(t *testing.T) {
	userID := domain.MustNewID()
	practiceID := domain.MustNewID()
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	attempt, err := NewQuizAttempt(userID, practiceID, true, now)
	if err != nil {
		t.Fatalf("NewQuizAttempt() error = %v", err)
	}
	if attempt.ID.IsZero() {
		t.Fatal("ID is zero, want a UUID")
	}
	if attempt.UserID != userID || attempt.PracticeID != practiceID {
		t.Fatalf("attempt ids = user %s practice %s", attempt.UserID, attempt.PracticeID)
	}
	if !attempt.Correct {
		t.Fatal("Correct = false, want true")
	}
	if !attempt.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %v, want %v", attempt.CreatedAt, now)
	}
}

func TestNewQuizAttempt_ZeroUserID_ReturnsValidationError(t *testing.T) {
	_, err := NewQuizAttempt(domain.ID{}, domain.MustNewID(), false, time.Now())
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.ValidationError", err)
	}
}

func TestNewQuizAttempt_ZeroPracticeID_ReturnsValidationError(t *testing.T) {
	_, err := NewQuizAttempt(domain.MustNewID(), domain.ID{}, false, time.Now())
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.ValidationError", err)
	}
}

func TestNewQuizAttempt_ZeroCreatedAt_ReturnsValidationError(t *testing.T) {
	_, err := NewQuizAttempt(domain.MustNewID(), domain.MustNewID(), false, time.Time{})
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("error = %v, want *domain.ValidationError", err)
	}
}

func TestNewQuizAttempt_IDGenerationError_Propagates(t *testing.T) {
	previous := generateID
	generateID = func() (domain.ID, error) { return domain.ID{}, errors.New("id failure") }
	defer func() { generateID = previous }()

	if _, err := NewQuizAttempt(domain.MustNewID(), domain.MustNewID(), false, time.Now()); err == nil {
		t.Fatal("NewQuizAttempt() error = nil, want error")
	}
}

func TestBuildQuizStats_MixedAttempts(t *testing.T) {
	stats := BuildQuizStats([]QuizAttempt{{Correct: true}, {Correct: true}, {Correct: false}})
	if stats.TotalAttempts != 3 || stats.CorrectAttempts != 2 {
		t.Fatalf("stats = %+v, want total=3 correct=2", stats)
	}
	if diff := stats.Accuracy - 2.0/3.0; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("Accuracy = %v, want %v", stats.Accuracy, 2.0/3.0)
	}
}

func TestBuildQuizStats_NoAttempts_AccuracyIsOne(t *testing.T) {
	stats := BuildQuizStats(nil)
	if stats.TotalAttempts != 0 || stats.CorrectAttempts != 0 {
		t.Fatalf("stats = %+v, want zero attempts", stats)
	}
	if stats.Accuracy != 1 {
		t.Fatalf("Accuracy = %v, want 1", stats.Accuracy)
	}
}
