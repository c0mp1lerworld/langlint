//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

func TestPostgresProgressRepository_ListSamplesByUser_CountsFragmentsAndErrors(t *testing.T) {
	resetDB(t)
	repo := repositories.NewProgressRepository(pool)
	ctx := context.Background()
	userID := domain.MustNewID()
	practiceID := savePractice(t, userID)

	createdAt := time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC)
	a, err := analysis.NewAnalysis(practiceID, "gpt-x", "2026-01", createdAt)
	require.NoError(t, err)
	fragments := []analysis.Fragment{
		{SourceES: "El perro corre", ErrorPatterns: []domain.ErrorPattern{{
			Code: domain.ErrorPatternCodeTenseAgreement, Severity: domain.ErrorPatternSeverityModerate,
		}}},
		{SourceES: "Él juega", ErrorPatterns: []domain.ErrorPattern{
			{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityMinor},
			{Code: domain.ErrorPatternCodeLexicalChoice, Severity: domain.ErrorPatternSeverityMinor},
		}},
	}
	require.NoError(t, a.Complete(fragments))
	require.NoError(t, repositories.NewAnalysisRepository(pool).Save(ctx, a))

	samples, err := repo.ListSamplesByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, samples, 1)
	require.Equal(t, 2, samples[0].TotalFragments)
	require.Equal(t, 3, samples[0].ErrorCount)
	require.WithinDuration(t, createdAt, samples[0].CompletedAt, time.Microsecond)
}

func TestPostgresProgressRepository_ListSamplesByUser_ExcludesOtherUsers(t *testing.T) {
	resetDB(t)
	repo := repositories.NewProgressRepository(pool)
	ctx := context.Background()

	other := mustCompletedAnalysis(t, savePractice(t, domain.MustNewID()))
	require.NoError(t, repositories.NewAnalysisRepository(pool).Save(ctx, other))

	samples, err := repo.ListSamplesByUser(ctx, domain.MustNewID())
	require.NoError(t, err)
	require.Empty(t, samples)
}
