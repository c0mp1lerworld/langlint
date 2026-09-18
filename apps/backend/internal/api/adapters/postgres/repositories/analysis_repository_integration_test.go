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

func mustCompletedAnalysis(t *testing.T, practiceID domain.ID) *analysis.Analysis {
	t.Helper()
	a, err := analysis.NewAnalysis(practiceID, "gpt-x", "2026-01", time.Now().UTC())
	require.NoError(t, err)
	fragment := analysis.Fragment{
		SourceES:             "El perro corre",
		UserDraft:            "The dog run",
		Correction:           "The dog runs",
		TargetVerbReview:     "run -> runs",
		LexicalClarification: "third person -s",
		GrammarExplanation:   "present simple agreement",
		ErrorPatterns: []domain.ErrorPattern{{
			Code:     domain.ErrorPatternCodeTenseAgreement,
			Severity: domain.ErrorPatternSeverityModerate,
			Note:     "missing -s",
		}},
	}
	require.NoError(t, a.Complete([]analysis.Fragment{fragment}))
	return a
}

func TestPostgresAnalysisRepository_Save_ThenGetByPracticeID_RoundTrip(t *testing.T) {
	resetDB(t)
	repo := repositories.NewAnalysisRepository(pool)
	ctx := context.Background()
	practiceID := savePractice(t, domain.MustNewID())
	a := mustCompletedAnalysis(t, practiceID)

	require.NoError(t, repo.Save(ctx, a))

	got, err := repo.GetByPracticeID(ctx, practiceID)
	require.NoError(t, err)
	require.Equal(t, a.ID, got.ID)
	require.Equal(t, practiceID, got.PracticeID)
	require.Equal(t, "gpt-x", got.Model)
	require.Equal(t, "2026-01", got.ModelVersion)
	require.Equal(t, analysis.AnalysisStatusCompleted, got.Status)
	require.Equal(t, a.Fragments, got.Fragments)
	require.WithinDuration(t, a.CreatedAt, got.CreatedAt, time.Microsecond)
}

func TestPostgresAnalysisRepository_GetByPracticeID_Missing_ReturnsNotFound(t *testing.T) {
	resetDB(t)
	repo := repositories.NewAnalysisRepository(pool)

	_, err := repo.GetByPracticeID(context.Background(), domain.MustNewID())

	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
	require.Equal(t, "analysis", notFound.Field)
}

func TestPostgresAnalysisRepository_Save_Update_PersistsCompletion(t *testing.T) {
	resetDB(t)
	repo := repositories.NewAnalysisRepository(pool)
	ctx := context.Background()
	practiceID := savePractice(t, domain.MustNewID())

	a, err := analysis.NewAnalysis(practiceID, "gpt-x", "2026-01", time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, a))

	require.NoError(t, a.Complete([]analysis.Fragment{{SourceES: "hola"}}))
	require.NoError(t, repo.Save(ctx, a))

	got, err := repo.GetByPracticeID(ctx, practiceID)
	require.NoError(t, err)
	require.Equal(t, analysis.AnalysisStatusCompleted, got.Status)
	require.Len(t, got.Fragments, 1)
}
