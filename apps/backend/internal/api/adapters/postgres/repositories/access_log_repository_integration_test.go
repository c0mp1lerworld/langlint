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

func newAccessEvent(t *testing.T, userID domain.ID, action, resourceType string, at time.Time) *identity.AccessEvent {
	t.Helper()
	event, err := identity.NewAccessEvent(userID, action, resourceType, "", at)
	require.NoError(t, err)
	return event
}

func TestPostgresAccessLogRepository_Append_ThenListByUser(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewAccessLogRepository(pool)
	userID := domain.MustNewID()
	base := time.Now().UTC().Truncate(time.Microsecond)

	older := newAccessEvent(t, userID, "GET", "/practices", base)
	newer := newAccessEvent(t, userID, "POST", "/practices", base.Add(time.Minute))
	require.NoError(t, repo.Append(ctx, older))
	require.NoError(t, repo.Append(ctx, newer))

	events, total, err := repo.ListByUser(ctx, userID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, events, 2)
	require.Equal(t, newer.ID, events[0].ID)
	require.Equal(t, "POST", events[0].Action)
	require.Equal(t, "/practices", events[0].ResourceType)
	require.WithinDuration(t, base.Add(time.Minute), events[0].OccurredAt, time.Microsecond)
}

func TestPostgresAccessLogRepository_ListByUser_PaginatesAndIsolates(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewAccessLogRepository(pool)
	owner, other := domain.MustNewID(), domain.MustNewID()
	base := time.Now().UTC()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Append(ctx, newAccessEvent(t, owner, "GET", "/practices", base.Add(time.Duration(i)*time.Second))))
	}
	require.NoError(t, repo.Append(ctx, newAccessEvent(t, other, "GET", "/practices", base)))

	page, total, err := repo.ListByUser(ctx, owner, 2, 0)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, page, 2)

	_, otherTotal, err := repo.ListByUser(ctx, other, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, otherTotal)
}

func TestPostgresAccessLogRepository_Append_EmptyResource_RoundTrips(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewAccessLogRepository(pool)
	userID := domain.MustNewID()

	require.NoError(t, repo.Append(ctx, newAccessEvent(t, userID, "DELETE", "", time.Now().UTC())))

	events, _, err := repo.ListByUser(ctx, userID, 10, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Empty(t, events[0].ResourceType)
	require.Empty(t, events[0].ResourceID)
}
