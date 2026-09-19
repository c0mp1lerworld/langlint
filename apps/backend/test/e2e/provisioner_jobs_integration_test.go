//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/pseudonymizer"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/testdb"
)

const day = 24 * time.Hour

// testPseudonymizer builds the HMAC that keys analytics rows in the tests (A8).
func testPseudonymizer(t *testing.T) *pseudonymizer.Pseudonymizer {
	t.Helper()
	p, err := pseudonymizer.New("test-secret")
	require.NoError(t, err)
	return p
}

func seedPractice(t *testing.T, pool *pgxpool.Pool, userID domain.ID, deletedAt *time.Time) domain.ID {
	t.Helper()
	id := domain.MustNewID()
	now := time.Now().UTC()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO practices (id, user_id, source_text, draft_text, target_rules, status, created_at, updated_at, deleted_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)`,
		id.String(), userID.String(), "texto", "draft", `[]`, "completed", now, now, deletedAt)
	require.NoError(t, err)
	return id
}

func seedAnalysis(t *testing.T, pool *pgxpool.Pool, practiceID domain.ID, status string, patterns []domain.ErrorPattern) {
	t.Helper()
	fragments, err := json.Marshal([]analysis.Fragment{{ErrorPatterns: patterns}})
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO analyses (id, practice_id, fragments, model, model_version, status, created_at)
		 VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7)`,
		domain.MustNewID().String(), practiceID.String(), string(fragments), "model", "v1", status, time.Now().UTC())
	require.NoError(t, err)
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	require.NoError(t, pool.QueryRow(context.Background(), query, args...).Scan(&count))
	return count
}

func startPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, cleanup, err := testdb.Start()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	return pool
}

func TestProvisionerRefreshAggregates_RebuildsFromSourceOfTruth(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()
	userID := domain.MustNewID()

	practiceID := seedPractice(t, pool, userID, nil)
	seedAnalysis(t, pool, practiceID, "completed", []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityMinor},
		{Code: domain.ErrorPatternCodeWordOrder, Severity: domain.ErrorPatternSeverityMinor},
		{Code: domain.ErrorPatternCodeTenseAgreement, Severity: domain.ErrorPatternSeverityModerate},
	})

	// A drifted metric that the reconciliation must overwrite.
	_, err := pool.Exec(ctx,
		`INSERT INTO error_metrics (user_id, code, "window", count, last_seen_at) VALUES ($1, $2, $3, $4, $5)`,
		userID.String(), "false_friend", "week", 99, time.Now().UTC())
	require.NoError(t, err)

	svc := services.NewRefreshAggregatesService(
		repositories.NewAnalyticsSourceRepository(pool),
		repositories.NewErrorMetricRepository(pool, testPseudonymizer(t)),
	)
	affected, err := svc.Refresh(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, affected) // 2 codes x 3 windows

	require.Equal(t, 0, countRows(t, pool,
		`SELECT count(*) FROM error_metrics WHERE code = 'false_friend'`))
	require.Equal(t, 3, countRows(t, pool,
		`SELECT count(*) FROM error_metrics WHERE code = 'word_order' AND count = 2`))
	require.Equal(t, 3, countRows(t, pool,
		`SELECT count(*) FROM error_metrics WHERE code = 'tense_agreement' AND count = 1`))
}

func TestProvisionerPurgeRawData_RemovesExpiredSoftDeleted(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()
	userID := domain.MustNewID()

	longAgo := time.Now().UTC().Add(-40 * day)
	expired := seedPractice(t, pool, userID, &longAgo)
	seedAnalysis(t, pool, expired, "completed", nil)
	active := seedPractice(t, pool, userID, nil)

	svc := services.NewPurgeRawDataService(repositories.NewRawDataRepository(pool), 30*day)
	purged, err := svc.Purge(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, purged)

	require.Equal(t, 0, countRows(t, pool, `SELECT count(*) FROM practices WHERE id = $1`, expired.String()))
	require.Equal(t, 0, countRows(t, pool, `SELECT count(*) FROM analyses WHERE practice_id = $1`, expired.String()))
	require.Equal(t, 1, countRows(t, pool, `SELECT count(*) FROM practices WHERE id = $1`, active.String()))
}

func TestProvisionerExecuteDeletions_PurgesAfterGrace(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()
	userID := domain.MustNewID()
	otherUserID := domain.MustNewID()

	practiceID := seedPractice(t, pool, userID, nil)
	seedAnalysis(t, pool, practiceID, "completed", nil)
	otherPractice := seedPractice(t, pool, otherUserID, nil)

	req, err := identity.NewDeletionRequest(userID, time.Now().UTC().Add(-31*day))
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO deletion_requests (id, user_id, requested_at, executed_at) VALUES ($1, $2, $3, $4)`,
		req.ID.String(), userID.String(), req.RequestedAt, req.ExecutedAt)
	require.NoError(t, err)

	svc := services.NewExecuteDeletionsService(repositories.NewDeletionRepository(pool, testPseudonymizer(t)), 30*day)
	executed, err := svc.Run(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, executed)

	require.Equal(t, 0, countRows(t, pool, `SELECT count(*) FROM practices WHERE user_id = $1`, userID.String()))
	require.Equal(t, 0, countRows(t, pool, `SELECT count(*) FROM analyses WHERE practice_id = $1`, practiceID.String()))
	require.Equal(t, 1, countRows(t, pool, `SELECT count(*) FROM practices WHERE user_id = $1`, otherUserID.String()))
	require.Equal(t, 1, countRows(t, pool,
		`SELECT count(*) FROM deletion_requests WHERE id = $1 AND executed_at IS NOT NULL`, req.ID.String()))
	require.Equal(t, 1, countRows(t, pool, `SELECT count(*) FROM practices WHERE id = $1`, otherPractice.String()))
}

func TestProvisionerExecuteDeletions_WithinGrace_KeepsData(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()
	userID := domain.MustNewID()

	practiceID := seedPractice(t, pool, userID, nil)
	req, err := identity.NewDeletionRequest(userID, time.Now().UTC().Add(-29*day))
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO deletion_requests (id, user_id, requested_at, executed_at) VALUES ($1, $2, $3, $4)`,
		req.ID.String(), userID.String(), req.RequestedAt, req.ExecutedAt)
	require.NoError(t, err)

	svc := services.NewExecuteDeletionsService(repositories.NewDeletionRepository(pool, testPseudonymizer(t)), 30*day)
	executed, err := svc.Run(ctx)
	require.NoError(t, err)
	require.Zero(t, executed)

	require.Equal(t, 1, countRows(t, pool, `SELECT count(*) FROM practices WHERE id = $1`, practiceID.String()))
}
