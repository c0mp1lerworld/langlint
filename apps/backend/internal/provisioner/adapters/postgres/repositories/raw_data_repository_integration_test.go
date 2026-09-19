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

func TestPostgresRawDataRepository_PurgePracticesDeletedBefore_RemovesOnlyExpired(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewRawDataRepository(pool)
	userID := domain.MustNewID()
	now := time.Now().UTC()
	cutoff := now.Add(-30 * 24 * time.Hour)

	activeID := insertPractice(t, userID, "draft", nil)
	activeAnalysisID := insertAnalysis(t, activeID, "completed", nil, now)

	longAgo := now.Add(-40 * 24 * time.Hour)
	expiredID := insertPractice(t, userID, "completed", &longAgo)
	expiredAnalysisID := insertAnalysis(t, expiredID, "completed", nil, now)

	yesterday := now.Add(-24 * time.Hour)
	freshID := insertPractice(t, userID, "completed", &yesterday)
	insertAnalysis(t, freshID, "completed", nil, now)

	purged, err := repo.PurgePracticesDeletedBefore(ctx, cutoff)
	require.NoError(t, err)
	require.Equal(t, 1, purged)

	require.False(t, practiceExists(t, expiredID))
	require.False(t, analysisExists(t, expiredAnalysisID))

	require.True(t, practiceExists(t, activeID))
	require.True(t, analysisExists(t, activeAnalysisID))
	require.True(t, practiceExists(t, freshID))
}

func TestPostgresRawDataRepository_PurgePracticesDeletedBefore_IsIdempotent(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewRawDataRepository(pool)
	userID := domain.MustNewID()
	now := time.Now().UTC()
	longAgo := now.Add(-40 * 24 * time.Hour)
	insertPractice(t, userID, "completed", &longAgo)

	first, err := repo.PurgePracticesDeletedBefore(ctx, now.Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, first)

	second, err := repo.PurgePracticesDeletedBefore(ctx, now.Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.Zero(t, second)
}
