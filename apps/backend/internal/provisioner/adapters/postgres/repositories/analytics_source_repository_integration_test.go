//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/adapters/postgres/repositories"
)

func TestPostgresAnalyticsSourceRepository_ListCompletedAnalyses_FiltersAndDecodes(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewAnalyticsSourceRepository(pool)
	userID := domain.MustNewID()
	now := time.Now().UTC().Truncate(time.Microsecond)

	completedPractice := insertPractice(t, userID, "completed", nil)
	insertAnalysis(t, completedPractice, "completed", []domain.ErrorPattern{{
		Code:     domain.ErrorPatternCodeWordOrder,
		Severity: domain.ErrorPatternSeverityMinor,
	}}, now)

	pendingPractice := insertPractice(t, userID, "analyzing", nil)
	insertAnalysis(t, pendingPractice, "pending", nil, now)

	expired := now.Add(-24 * time.Hour)
	deletedPractice := insertPractice(t, userID, "completed", &expired)
	insertAnalysis(t, deletedPractice, "completed", nil, now)

	got, err := repo.ListCompletedAnalyses(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, userID, got[0].UserID)
	require.Len(t, got[0].Fragments, 1)
	require.Equal(t, domain.ErrorPatternCodeWordOrder, got[0].Fragments[0].ErrorPatterns[0].Code)
	require.WithinDuration(t, now, got[0].CompletedAt, time.Microsecond)
}

func TestPostgresAnalyticsSourceRepository_ListCompletedAnalyses_Empty(t *testing.T) {
	resetDB(t)
	repo := repositories.NewAnalyticsSourceRepository(pool)

	got, err := repo.ListCompletedAnalyses(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}
