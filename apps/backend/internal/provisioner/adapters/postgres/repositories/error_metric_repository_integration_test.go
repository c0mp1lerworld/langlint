//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/adapters/postgres/repositories"
)

func TestPostgresErrorMetricRepository_ReplaceAll_RewritesTable(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewErrorMetricRepository(pool)
	userID := domain.MustNewID()
	now := time.Now().UTC()
	insertErrorMetric(t, userID, "word_order", "week", 5, now)

	metric, err := analytics.NewErrorMetric(userID, domain.ErrorPatternCodeTenseAgreement, analytics.WindowWeek, now)
	require.NoError(t, err)
	metric.Count = 7

	require.NoError(t, repo.ReplaceAll(ctx, []*analytics.ErrorMetric{metric}))

	rows := listErrorMetrics(t, userID)
	require.Len(t, rows, 1)
	require.Equal(t, "tense_agreement", rows[0].code)
	require.Equal(t, 7, rows[0].count)
}

func TestPostgresErrorMetricRepository_ReplaceAll_EmptyWipesTable(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewErrorMetricRepository(pool)
	userID := domain.MustNewID()
	insertErrorMetric(t, userID, "word_order", "week", 5, time.Now().UTC())

	require.NoError(t, repo.ReplaceAll(ctx, nil))

	require.Empty(t, listErrorMetrics(t, userID))
}

func TestPostgresErrorMetricRepository_ReplaceAll_MultipleWindowsAndUsers(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewErrorMetricRepository(pool)
	first, second := domain.MustNewID(), domain.MustNewID()
	now := time.Now().UTC()

	metrics := make([]*analytics.ErrorMetric, 0)
	for _, user := range []domain.ID{first, second} {
		for _, window := range analytics.AllWindows {
			m, err := analytics.NewErrorMetric(user, domain.ErrorPatternCodeWordOrder, window, now)
			require.NoError(t, err)
			metrics = append(metrics, m)
		}
	}

	require.NoError(t, repo.ReplaceAll(ctx, metrics))

	require.Len(t, listErrorMetrics(t, first), 3)
	require.Len(t, listErrorMetrics(t, second), 3)
}
