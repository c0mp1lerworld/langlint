//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestPostgresOutbox_Append_OutsideTransaction(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	outbox := postgres.NewOutbox(pool)

	require.NoError(t, outbox.Append(ctx, domain.PracticeCreated{
		PracticeID: domain.MustNewID(),
		UserID:     domain.MustNewID(),
		Version:    1,
	}))

	var eventType string
	require.NoError(t, pool.QueryRow(ctx, `SELECT event_type FROM outbox_events`).Scan(&eventType))
	require.Equal(t, domain.EventNamePracticeCreated, eventType)
}

func TestPostgresOutbox_Append_InsideTransaction(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	outbox := postgres.NewOutbox(pool)
	uow := postgres.NewUnitOfWork(pool)

	require.NoError(t, uow.InTransaction(ctx, func(txCtx context.Context) error {
		return outbox.Append(txCtx, domain.AnalysisFailed{
			AnalysisID: domain.MustNewID(),
			PracticeID: domain.MustNewID(),
			Reason:     "timeout",
			Version:    1,
		})
	}))

	var eventType string
	require.NoError(t, pool.QueryRow(ctx, `SELECT event_type FROM outbox_events`).Scan(&eventType))
	require.Equal(t, domain.EventNameAnalysisFailed, eventType)
}
