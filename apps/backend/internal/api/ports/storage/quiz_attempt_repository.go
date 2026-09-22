package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// QuizAttemptRepository persists the append-only quiz attempts ledger (checklist
// 9.7). It is keyed by the raw portable user id (owned data, A9), unlike the
// pseudonymized error metrics.
type QuizAttemptRepository interface {
	Append(ctx context.Context, attempt *analytics.QuizAttempt) error
	ListByUser(ctx context.Context, userID domain.ID) ([]analytics.QuizAttempt, error)
}
