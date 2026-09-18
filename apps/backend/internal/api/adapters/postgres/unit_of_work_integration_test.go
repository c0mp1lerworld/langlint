//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/testsupport"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, cleanup, err := testsupport.Start()
	if err != nil {
		panic(err)
	}
	pool = p
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestPostgresUnitOfWork_InTransaction_CommitPersists(t *testing.T) {
	resetDB(t)
	uow := postgres.NewUnitOfWork(pool)
	repo := repositories.NewPracticeRepository(pool)
	p := newPractice(t, domain.MustNewID())

	err := uow.InTransaction(context.Background(), func(txCtx context.Context) error {
		return repo.Save(txCtx, p)
	})
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	require.Equal(t, p.ID, got.ID)
}

func TestPostgresUnitOfWork_InTransaction_ErrorRollsBack(t *testing.T) {
	resetDB(t)
	uow := postgres.NewUnitOfWork(pool)
	repo := repositories.NewPracticeRepository(pool)
	p := newPractice(t, domain.MustNewID())

	boom := errors.New("boom")
	err := uow.InTransaction(context.Background(), func(txCtx context.Context) error {
		if err := repo.Save(txCtx, p); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	var count int
	err = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM practices WHERE id = $1", p.ID.String(),
	).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}

func resetDB(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE practices, analyses, error_metrics, outbox_events")
	require.NoError(t, err)
}

func newPractice(t *testing.T, userID domain.ID) *practice.Practice {
	t.Helper()
	source, err := practice.NewSourceText("El perro corre")
	require.NoError(t, err)
	draft, err := practice.NewDraftText("The dog run")
	require.NoError(t, err)
	rule, err := practice.NewTargetRule("run", "present_simple", "")
	require.NoError(t, err)
	p, err := practice.NewPractice(userID, source, draft, []practice.TargetRule{rule}, time.Now().UTC())
	require.NoError(t, err)
	return p
}
