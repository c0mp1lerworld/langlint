//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/services"
)

// TestPrivacy_RawDataSeparatedFromAnalytics certifies the A8 separation: the
// study text (raw PII) lives only in the dedicated raw tables (practices,
// analyses), while analytics (error_metrics) carries a pseudonymized key and no
// free text. Purging/forgetting the user removes every trace, including the
// pseudonymized metrics.
func TestPrivacy_RawDataSeparatedFromAnalytics(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()
	userID := domain.MustNewID()
	pseudonyms := testPseudonymizer(t)
	now := time.Now().UTC()
	const rawText = "María escribió a maria@example.com desde Barcelona."

	// Raw study content is stored in the dedicated raw tables.
	practiceID := seedPractice(t, pool, userID, nil)
	_, err := pool.Exec(ctx, `UPDATE practices SET source_text = $2 WHERE id = $1`, practiceID.String(), rawText)
	require.NoError(t, err)
	seedAnalysis(t, pool, practiceID, "completed", []domain.ErrorPattern{
		{Code: domain.ErrorPatternCodeTenseAgreement, Severity: domain.ErrorPatternSeverityMinor},
	})
	rawFragments, err := json.Marshal([]analysis.Fragment{{SourceES: rawText, UserDraft: "Maria wrote...", Correction: "..."}})
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`UPDATE analyses SET fragments = $2::jsonb WHERE practice_id = $1`,
		practiceID.String(), string(rawFragments))
	require.NoError(t, err)

	// Analytics materialize the pseudonymized key, never the raw id.
	_, err = pool.Exec(ctx,
		`INSERT INTO error_metrics (user_id, code, "window", count, last_seen_at) VALUES ($1, $2, $3, $4, $5)`,
		pseudonyms.Pseudonymize(userID.String()), "tense_agreement", "week", 1, now)
	require.NoError(t, err)

	require.Equal(t, 1, countRows(t, pool,
		`SELECT count(*) FROM practices WHERE id = $1 AND source_text = $2`, practiceID.String(), rawText))
	require.Equal(t, 1, countRows(t, pool,
		`SELECT count(*) FROM analyses WHERE practice_id = $1 AND fragments::text LIKE '%maria@example.com%'`, practiceID.String()))

	require.Equal(t, 1, countRows(t, pool,
		`SELECT count(*) FROM error_metrics WHERE user_id = $1`, pseudonyms.Pseudonymize(userID.String())))
	require.Equal(t, 0, countRows(t, pool,
		`SELECT count(*) FROM error_metrics WHERE user_id = $1`, userID.String()))

	// Quiz attempts are owned data (raw user id, A9) and are purged too (9.7).
	_, err = pool.Exec(ctx,
		`INSERT INTO quiz_attempts (id, user_id, practice_id, correct, created_at) VALUES ($1, $2, $3, $4, $5)`,
		domain.MustNewID().String(), userID.String(), practiceID.String(), true, now)
	require.NoError(t, err)
	require.Equal(t, 1, countRows(t, pool,
		`SELECT count(*) FROM quiz_attempts WHERE user_id = $1`, userID.String()))

	// The right to be forgotten removes the raw data and the pseudonymized
	// analytics of the user.
	req, err := identity.NewDeletionRequest(userID, now.Add(-31*day))
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO deletion_requests (id, user_id, requested_at, executed_at) VALUES ($1, $2, $3, $4)`,
		req.ID.String(), userID.String(), req.RequestedAt, req.ExecutedAt)
	require.NoError(t, err)

	svc := services.NewExecuteDeletionsService(repositories.NewDeletionRepository(pool, pseudonyms), 30*day)
	executed, err := svc.Run(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, executed)

	require.Equal(t, 0, countRows(t, pool, `SELECT count(*) FROM practices WHERE id = $1`, practiceID.String()))
	require.Equal(t, 0, countRows(t, pool, `SELECT count(*) FROM analyses WHERE practice_id = $1`, practiceID.String()))
	require.Equal(t, 0, countRows(t, pool,
		`SELECT count(*) FROM error_metrics WHERE user_id = $1`, pseudonyms.Pseudonymize(userID.String())))
	require.Equal(t, 0, countRows(t, pool,
		`SELECT count(*) FROM quiz_attempts WHERE user_id = $1`, userID.String()))
}
