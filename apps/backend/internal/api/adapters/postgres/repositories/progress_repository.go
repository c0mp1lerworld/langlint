package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// PostgresProgressRepository derives the progress series samples from the
// append-only analyses of non-deleted practices (A4). It stores nothing: the
// metrics are computed on read, so they can never drift.
type PostgresProgressRepository struct {
	pool *pgxpool.Pool
}

var _ storage.ProgressRepository = (*PostgresProgressRepository)(nil)

// NewProgressRepository builds the repository over the given pool.
func NewProgressRepository(pool *pgxpool.Pool) *PostgresProgressRepository {
	return &PostgresProgressRepository{pool: pool}
}

// ListSamplesByUser returns one sample per completed analysis of the user,
// ordered chronologically. The user id filters the raw source of truth and is
// never materialized (A8).
func (r *PostgresProgressRepository) ListSamplesByUser(ctx context.Context, userID domain.ID) ([]analytics.ProgressSample, error) {
	const q = `
SELECT a.created_at, a.fragments
FROM analyses a
JOIN practices p ON p.id = a.practice_id
WHERE a.status = $1 AND p.deleted_at IS NULL AND p.user_id = $2
ORDER BY a.created_at, a.id`

	rows, err := conn(ctx, r.pool).Query(ctx, q, analysis.AnalysisStatusCompleted.String(), userID.String())
	if err != nil {
		return nil, mapError("progress", err)
	}
	defer rows.Close()

	samples := make([]analytics.ProgressSample, 0)
	for rows.Next() {
		var (
			createdAt time.Time
			fragments []byte
		)
		if err := rows.Scan(&createdAt, &fragments); err != nil {
			return nil, mapError("progress", err)
		}

		var parsed []analysis.Fragment
		if err := json.Unmarshal(fragments, &parsed); err != nil {
			return nil, mapError("progress", fmt.Errorf("decode fragments: %w", err))
		}

		samples = append(samples, analytics.ProgressSample{
			CompletedAt:    createdAt,
			TotalFragments: len(parsed),
			ErrorCount:     countErrorPatterns(parsed),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("progress", err)
	}

	return samples, nil
}

// countErrorPatterns counts every error pattern across the fragments.
func countErrorPatterns(fragments []analysis.Fragment) int {
	total := 0
	for _, fragment := range fragments {
		total += len(fragment.ErrorPatterns)
	}
	return total
}
