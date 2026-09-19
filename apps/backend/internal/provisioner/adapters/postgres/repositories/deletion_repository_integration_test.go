//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/adapters/postgres/repositories"
)

func TestPostgresDeletionRepository_ListPending_OnlyUnexecuted(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewDeletionRepository(pool, testPseudonyms)
	userID := domain.MustNewID()
	now := time.Now().UTC()

	pendingID := insertDeletionRequest(t, userID, now.Add(-31*24*time.Hour))
	executedID := insertDeletionRequest(t, userID, now.Add(-60*24*time.Hour))
	_, err := pool.Exec(ctx, `UPDATE deletion_requests SET executed_at = $2 WHERE id = $1`, executedID.String(), now)
	require.NoError(t, err)

	got, err := repo.ListPending(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, pendingID, got[0].ID)
	require.Equal(t, userID, got[0].UserID)
	require.Nil(t, got[0].ExecutedAt)
}

func TestPostgresDeletionRepository_Execute_PurgesUserAndMarksExecuted(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewDeletionRepository(pool, testPseudonyms)
	userID := domain.MustNewID()
	otherUserID := domain.MustNewID()
	now := time.Now().UTC()

	practiceID := insertPractice(t, userID, "completed", nil)
	analysisID := insertAnalysis(t, practiceID, "completed", nil, now)
	insertErrorMetric(t, userID, "word_order", "week", 3, now)

	otherPracticeID := insertPractice(t, otherUserID, "draft", nil)
	otherAnalysisID := insertAnalysis(t, otherPracticeID, "completed", nil, now)

	req, err := identity.NewDeletionRequest(userID, now.Add(-31*24*time.Hour))
	require.NoError(t, err)
	require.NoError(t, insertPendingRequest(ctx, *req))
	require.NoError(t, req.MarkExecuted(now))

	require.NoError(t, repo.Execute(ctx, req))

	require.False(t, practiceExists(t, practiceID))
	require.False(t, analysisExists(t, analysisID))
	require.Empty(t, listErrorMetrics(t, userID))
	require.True(t, deletionRequestExecuted(t, req.ID))

	require.True(t, practiceExists(t, otherPracticeID))
	require.True(t, analysisExists(t, otherAnalysisID))
}

func TestPostgresDeletionRepository_Execute_RequiresExecutedAt(t *testing.T) {
	resetDB(t)
	repo := repositories.NewDeletionRepository(pool, testPseudonyms)

	pending, err := identity.NewDeletionRequest(domain.MustNewID(), time.Now().UTC())
	require.NoError(t, err)

	err = repo.Execute(context.Background(), pending)
	var invalidState *domain.InvalidStateError
	require.ErrorAs(t, err, &invalidState)
}

// insertPendingRequest persists a domain deletion request as-is.
func insertPendingRequest(ctx context.Context, req identity.DeletionRequest) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO deletion_requests (id, user_id, requested_at, executed_at) VALUES ($1, $2, $3, $4)`,
		req.ID.String(), req.UserID.String(), req.RequestedAt, req.ExecutedAt)
	return err
}
