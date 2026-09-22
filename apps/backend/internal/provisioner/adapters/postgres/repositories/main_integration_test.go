//go:build integration

package repositories_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/testdb"
)

var pool *pgxpool.Pool

// testPseudonyms keys the analytics rows the tests seed, mirroring production
// (A8).
var testPseudonyms = mustPseudonymizer()

func mustPseudonymizer() *pseudonymizer.Pseudonymizer {
	p, err := pseudonymizer.New("test-secret")
	if err != nil {
		panic(err)
	}
	return p
}

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
		"TRUNCATE practices, analyses, error_metrics, outbox_events, deletion_requests, quiz_attempts")
	require.NoError(t, err)
}

func insertPractice(t *testing.T, userID domain.ID, status string, deletedAt *time.Time) domain.ID {
	t.Helper()
	id := domain.MustNewID()
	now := time.Now().UTC()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO practices (id, user_id, source_text, draft_text, target_rules, status, created_at, updated_at, deleted_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)`,
		id.String(), userID.String(), "texto base", "draft text", `[]`, status, now, now, deletedAt)
	require.NoError(t, err)
	return id
}

func insertAnalysis(t *testing.T, practiceID domain.ID, status string, patterns []domain.ErrorPattern, createdAt time.Time) domain.ID {
	t.Helper()
	id := domain.MustNewID()
	fragments := []analysis.Fragment{{ErrorPatterns: patterns}}
	raw, err := json.Marshal(fragments)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO analyses (id, practice_id, fragments, model, model_version, status, created_at)
		 VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7)`,
		id.String(), practiceID.String(), string(raw), "model", "version", status, createdAt)
	require.NoError(t, err)
	return id
}

func insertErrorMetric(t *testing.T, userID domain.ID, code, window string, count int, lastSeen time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO error_metrics (user_id, code, "window", count, last_seen_at) VALUES ($1, $2, $3, $4, $5)`,
		testPseudonyms.Pseudonymize(userID.String()), code, window, count, lastSeen)
	require.NoError(t, err)
}

func insertDeletionRequest(t *testing.T, userID domain.ID, requestedAt time.Time) domain.ID {
	t.Helper()
	id := domain.MustNewID()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO deletion_requests (id, user_id, requested_at) VALUES ($1, $2, $3)`,
		id.String(), userID.String(), requestedAt)
	require.NoError(t, err)
	return id
}

func practiceExists(t *testing.T, id domain.ID) bool {
	t.Helper()
	var exists bool
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM practices WHERE id = $1)`, id.String()).Scan(&exists))
	return exists
}

func analysisExists(t *testing.T, id domain.ID) bool {
	t.Helper()
	var exists bool
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM analyses WHERE id = $1)`, id.String()).Scan(&exists))
	return exists
}

func deletionRequestExecuted(t *testing.T, id domain.ID) bool {
	t.Helper()
	var executed bool
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT executed_at IS NOT NULL FROM deletion_requests WHERE id = $1`, id.String()).Scan(&executed))
	return executed
}

type metricRow struct {
	code  string
	count int
}

func listErrorMetrics(t *testing.T, userID domain.ID) []metricRow {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT code, count FROM error_metrics WHERE user_id = $1 ORDER BY code`, testPseudonyms.Pseudonymize(userID.String()))
	require.NoError(t, err)
	defer rows.Close()

	out := make([]metricRow, 0)
	for rows.Next() {
		var m metricRow
		require.NoError(t, rows.Scan(&m.code, &m.count))
		out = append(out, m)
	}
	require.NoError(t, rows.Err())
	return out
}
