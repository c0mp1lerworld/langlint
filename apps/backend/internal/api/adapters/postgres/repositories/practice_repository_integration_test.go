//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

func TestPostgresPracticeRepository_Save_ThenGetByID_RoundTrip(t *testing.T) {
	resetDB(t)
	repo := repositories.NewPracticeRepository(pool)
	ctx := context.Background()
	p := mustPractice(t, domain.MustNewID(), time.Now().UTC())

	require.NoError(t, repo.Save(ctx, p))

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, p.ID, got.ID)
	require.Equal(t, p.UserID, got.UserID)
	require.Equal(t, p.SourceText, got.SourceText)
	require.Equal(t, p.DraftText, got.DraftText)
	require.Equal(t, p.TargetRules, got.TargetRules)
	require.Equal(t, practice.PracticeStatusDraft, got.Status)
	require.WithinDuration(t, p.CreatedAt, got.CreatedAt, time.Microsecond)
	require.WithinDuration(t, p.UpdatedAt, got.UpdatedAt, time.Microsecond)
	require.Nil(t, got.DeletedAt)
}

func TestPostgresPracticeRepository_GetByID_Missing_ReturnsNotFound(t *testing.T) {
	resetDB(t)
	repo := repositories.NewPracticeRepository(pool)

	_, err := repo.GetByID(context.Background(), domain.MustNewID())

	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
	require.Equal(t, "practice", notFound.Field)
}

func TestPostgresPracticeRepository_Save_Update_PersistsState(t *testing.T) {
	resetDB(t)
	repo := repositories.NewPracticeRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	p := mustPractice(t, domain.MustNewID(), now)
	require.NoError(t, repo.Save(ctx, p))

	require.NoError(t, p.StartAnalysis(now.Add(time.Minute)))
	require.NoError(t, repo.Save(ctx, p))

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, practice.PracticeStatusAnalyzing, got.Status)
	require.WithinDuration(t, p.UpdatedAt, got.UpdatedAt, time.Microsecond)
}

func TestPostgresPracticeRepository_Delete_ExcludedFromReads(t *testing.T) {
	resetDB(t)
	repo := repositories.NewPracticeRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	p := mustPractice(t, domain.MustNewID(), now)
	require.NoError(t, repo.Save(ctx, p))

	require.NoError(t, p.Delete(now.Add(time.Minute)))
	require.NoError(t, repo.Save(ctx, p))

	_, err := repo.GetByID(ctx, p.ID)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)

	items, total, err := repo.ListByUser(ctx, p.UserID, 10, 0)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, items)
}

func TestPostgresPracticeRepository_ListByUser_PaginatesAndCounts(t *testing.T) {
	resetDB(t)
	repo := repositories.NewPracticeRepository(pool)
	ctx := context.Background()
	userID := domain.MustNewID()
	base := time.Now().UTC()
	p0 := mustPractice(t, userID, base)
	p1 := mustPractice(t, userID, base.Add(time.Second))
	p2 := mustPractice(t, userID, base.Add(2*time.Second))
	for _, p := range []*practice.Practice{p0, p1, p2} {
		require.NoError(t, repo.Save(ctx, p))
	}

	first, total, err := repo.ListByUser(ctx, userID, 2, 0)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, first, 2)
	require.Equal(t, p2.ID, first[0].ID)
	require.Equal(t, p1.ID, first[1].ID)

	second, total, err := repo.ListByUser(ctx, userID, 2, 2)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, second, 1)
	require.Equal(t, p0.ID, second[0].ID)
}

func TestPostgresPracticeRepository_ListByUser_IsolatesUsers(t *testing.T) {
	resetDB(t)
	repo := repositories.NewPracticeRepository(pool)
	ctx := context.Background()
	owner, other := domain.MustNewID(), domain.MustNewID()
	require.NoError(t, repo.Save(ctx, mustPractice(t, owner, time.Now().UTC())))
	require.NoError(t, repo.Save(ctx, mustPractice(t, other, time.Now().UTC())))

	items, total, err := repo.ListByUser(ctx, owner, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, owner, items[0].UserID)
}
