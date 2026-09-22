//go:build integration

package repositories_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/testdb"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, cleanup, err := testdb.Start()
	if err != nil {
		panic(err)
	}
	pool = p
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func resetDB(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"TRUNCATE practices, analyses, error_metrics, outbox_events, deletion_requests, access_events, study_sessions")
	require.NoError(t, err)
}

// testPseudonymizer builds the HMAC the repositories use to key analytics (A8).
func testPseudonymizer(t *testing.T) *pseudonymizer.Pseudonymizer {
	t.Helper()
	p, err := pseudonymizer.New("test-secret")
	require.NoError(t, err)
	return p
}

// storedMetricUserKey returns the analytics key of the only error metric.
func storedMetricUserKey(t *testing.T) string {
	t.Helper()
	var key string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT user_id FROM error_metrics LIMIT 1`).Scan(&key))
	return key
}

func mustPractice(t *testing.T, userID domain.ID, createdAt time.Time) *practice.Practice {
	t.Helper()
	source, err := practice.NewSourceText("El perro corre en el parque")
	require.NoError(t, err)
	draft, err := practice.NewDraftText("The dog run in the park")
	require.NoError(t, err)
	rule, err := practice.NewTargetRule("run", "present_simple", "irregular")
	require.NoError(t, err)
	p, err := practice.NewPractice(userID, source, draft, []practice.TargetRule{rule}, createdAt)
	require.NoError(t, err)
	return p
}

func savePractice(t *testing.T, userID domain.ID) domain.ID {
	t.Helper()
	p := mustPractice(t, userID, time.Now().UTC())
	require.NoError(t, repositories.NewPracticeRepository(pool).Save(context.Background(), p))
	return p.ID
}
