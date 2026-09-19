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

func TestPostgresErrorMetricRepository_Upsert_ReplacesAbsoluteCount(t *testing.T) {
	resetDB(t)
	repo := repositories.NewErrorMetricRepository(pool, testPseudonymizer(t))
	ctx := context.Background()
	userID := domain.MustNewID()
	now := time.Now().UTC()

	m, err := analytics.NewErrorMetric(userID, domain.ErrorPatternCodeWordOrder, analytics.WindowWeek, now)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, m))

	m.Record(now.Add(time.Minute))
	require.Equal(t, 2, m.Count)
	require.NoError(t, repo.Upsert(ctx, m))

	got, err := repo.ListByUser(ctx, userID, analytics.WindowWeek)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, 2, got[0].Count)
	require.Equal(t, domain.ErrorPatternCodeWordOrder, got[0].Code)
	require.WithinDuration(t, now.Add(time.Minute), got[0].LastSeenAt, time.Microsecond)
}

// TestPostgresErrorMetricRepository_Upsert_StoresPseudonym asserts analytics
// never materialize the raw user id (A8): the column holds the HMAC, not the
// UUID, and the raw id is not queryable.
func TestPostgresErrorMetricRepository_Upsert_StoresPseudonym(t *testing.T) {
	resetDB(t)
	pseudonyms := testPseudonymizer(t)
	repo := repositories.NewErrorMetricRepository(pool, pseudonyms)
	ctx := context.Background()
	userID := domain.MustNewID()
	now := time.Now().UTC()

	m, err := analytics.NewErrorMetric(userID, domain.ErrorPatternCodeWordOrder, analytics.WindowWeek, now)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, m))

	stored := storedMetricUserKey(t)
	require.Equal(t, pseudonyms.Pseudonymize(userID.String()), stored)
	require.NotEqual(t, userID.String(), stored)
}

func TestPostgresErrorMetricRepository_ListByUser_FiltersByWindow(t *testing.T) {
	resetDB(t)
	repo := repositories.NewErrorMetricRepository(pool, testPseudonymizer(t))
	ctx := context.Background()
	userID := domain.MustNewID()
	now := time.Now().UTC()

	day, err := analytics.NewErrorMetric(userID, domain.ErrorPatternCodeWordOrder, analytics.WindowDay, now)
	require.NoError(t, err)
	week, err := analytics.NewErrorMetric(userID, domain.ErrorPatternCodeFalseFriend, analytics.WindowWeek, now)
	require.NoError(t, err)
	other, err := analytics.NewErrorMetric(domain.MustNewID(), domain.ErrorPatternCodeWordOrder, analytics.WindowWeek, now)
	require.NoError(t, err)
	for _, m := range []*analytics.ErrorMetric{day, week, other} {
		require.NoError(t, repo.Upsert(ctx, m))
	}

	got, err := repo.ListByUser(ctx, userID, analytics.WindowWeek)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, domain.ErrorPatternCodeFalseFriend, got[0].Code)
}
