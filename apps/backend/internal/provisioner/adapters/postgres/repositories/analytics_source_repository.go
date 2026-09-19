package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
)

// PostgresAnalyticsSourceRepository reads the append-only analyses that feed
// the analytics reconciliation (A4).
type PostgresAnalyticsSourceRepository struct {
	pool *pgxpool.Pool
}

var _ storage.AnalyticsSourceRepository = (*PostgresAnalyticsSourceRepository)(nil)

// NewAnalyticsSourceRepository builds the repository over the given pool.
func NewAnalyticsSourceRepository(pool *pgxpool.Pool) *PostgresAnalyticsSourceRepository {
	return &PostgresAnalyticsSourceRepository{pool: pool}
}

// ListCompletedAnalyses returns the completed analyses of non-deleted practices,
// joined with their owner. The practice's user id is the metric key, so the
// analytics context never imports practice (A3).
func (r *PostgresAnalyticsSourceRepository) ListCompletedAnalyses(ctx context.Context) ([]storage.CompletedAnalysis, error) {
	const q = `
SELECT p.user_id, a.fragments, a.created_at
FROM analyses a
JOIN practices p ON p.id = a.practice_id
WHERE a.status = $1 AND p.deleted_at IS NULL
ORDER BY a.created_at, a.id`

	rows, err := r.pool.Query(ctx, q, analysis.AnalysisStatusCompleted.String())
	if err != nil {
		return nil, mapError("analytics_source", err)
	}
	defer rows.Close()

	analyses := make([]storage.CompletedAnalysis, 0)
	for rows.Next() {
		var (
			userID    string
			fragments []byte
			createdAt time.Time
		)
		if err := rows.Scan(&userID, &fragments, &createdAt); err != nil {
			return nil, mapError("analytics_source", err)
		}

		parsedUserID, err := domain.ParseID(userID)
		if err != nil {
			return nil, mapError("analytics_source", fmt.Errorf("decode user id: %w", err))
		}

		var parsedFragments []analysis.Fragment
		if err := json.Unmarshal(fragments, &parsedFragments); err != nil {
			return nil, mapError("analytics_source", fmt.Errorf("decode fragments: %w", err))
		}

		analyses = append(analyses, storage.CompletedAnalysis{
			UserID:      parsedUserID,
			Fragments:   parsedFragments,
			CompletedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("analytics_source", err)
	}

	return analyses, nil
}
