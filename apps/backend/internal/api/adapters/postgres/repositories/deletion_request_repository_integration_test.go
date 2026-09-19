//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
)

func TestPostgresDeletionRequestRepository_Append_ThenHasPending(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewDeletionRequestRepository(pool)
	userID := domain.MustNewID()

	pending, err := repo.HasPending(ctx, userID)
	require.NoError(t, err)
	require.False(t, pending)

	req, err := identity.NewDeletionRequest(userID, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Append(ctx, req))

	pending, err = repo.HasPending(ctx, userID)
	require.NoError(t, err)
	require.True(t, pending)
}

func TestPostgresDeletionRequestRepository_HasPending_IgnoresExecuted(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewDeletionRequestRepository(pool)
	userID := domain.MustNewID()

	req, err := identity.NewDeletionRequest(userID, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Append(ctx, req))

	_, err = pool.Exec(ctx, `UPDATE deletion_requests SET executed_at = now() WHERE id = $1`, req.ID.String())
	require.NoError(t, err)

	pending, err := repo.HasPending(ctx, userID)
	require.NoError(t, err)
	require.False(t, pending)
}

func TestPostgresDeletionRequestRepository_HasPending_IsolatesUsers(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewDeletionRequestRepository(pool)
	owner, other := domain.MustNewID(), domain.MustNewID()

	req, err := identity.NewDeletionRequest(owner, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Append(ctx, req))

	pending, err := repo.HasPending(ctx, other)
	require.NoError(t, err)
	require.False(t, pending)
}
