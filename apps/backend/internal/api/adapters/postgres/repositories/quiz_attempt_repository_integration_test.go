//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func newQuizAttempt(t *testing.T, userID, practiceID domain.ID, correct bool, at time.Time) *analytics.QuizAttempt {
	t.Helper()
	attempt, err := analytics.NewQuizAttempt(userID, practiceID, correct, at)
	require.NoError(t, err)
	return attempt
}

func TestPostgresQuizAttemptRepository_Append_ThenListByUser(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewQuizAttemptRepository(pool)
	userID := domain.MustNewID()
	practiceID := domain.MustNewID()
	base := time.Now().UTC().Truncate(time.Microsecond)

	older := newQuizAttempt(t, userID, practiceID, false, base)
	newer := newQuizAttempt(t, userID, practiceID, true, base.Add(time.Minute))
	require.NoError(t, repo.Append(ctx, older))
	require.NoError(t, repo.Append(ctx, newer))

	attempts, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, attempts, 2)
	require.Equal(t, newer.ID, attempts[0].ID)
	require.True(t, attempts[0].Correct)
	require.Equal(t, userID, attempts[0].UserID)
	require.Equal(t, practiceID, attempts[0].PracticeID)
}

func TestPostgresQuizAttemptRepository_ListByUser_IsolatesUsers(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewQuizAttemptRepository(pool)
	owner, other := domain.MustNewID(), domain.MustNewID()
	practiceID := domain.MustNewID()

	require.NoError(t, repo.Append(ctx, newQuizAttempt(t, owner, practiceID, true, time.Now().UTC())))
	require.NoError(t, repo.Append(ctx, newQuizAttempt(t, other, practiceID, false, time.Now().UTC())))

	attempts, err := repo.ListByUser(ctx, owner)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	require.Equal(t, owner, attempts[0].UserID)
}
