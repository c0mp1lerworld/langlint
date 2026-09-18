package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// PostgresAnalysisRepository implements storage.AnalysisRepository.
type PostgresAnalysisRepository struct {
	pool *pgxpool.Pool
}

var _ storage.AnalysisRepository = (*PostgresAnalysisRepository)(nil)

// NewAnalysisRepository builds a repository over the given pool.
func NewAnalysisRepository(pool *pgxpool.Pool) *PostgresAnalysisRepository {
	return &PostgresAnalysisRepository{pool: pool}
}

const analysisColumns = `id, practice_id, fragments, model, model_version, status, created_at`

// Save upserts the analysis, matching the aggregate's identity by id.
func (r *PostgresAnalysisRepository) Save(ctx context.Context, a *analysis.Analysis) error {
	fragments, err := json.Marshal(a.Fragments)
	if err != nil {
		return &domain.InternalError{Field: "fragments", Message: "cannot encode fragments"}
	}

	const q = `
INSERT INTO analyses (` + analysisColumns + `)
VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
	practice_id = EXCLUDED.practice_id,
	fragments = EXCLUDED.fragments,
	model = EXCLUDED.model,
	model_version = EXCLUDED.model_version,
	status = EXCLUDED.status`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		a.ID.String(),
		a.PracticeID.String(),
		string(fragments),
		a.Model,
		a.ModelVersion,
		a.Status.String(),
		a.CreatedAt,
	); err != nil {
		return mapError("analysis", err)
	}
	return nil
}

// GetByPracticeID returns the analysis of a practice (one-to-one).
func (r *PostgresAnalysisRepository) GetByPracticeID(ctx context.Context, practiceID domain.ID) (*analysis.Analysis, error) {
	const q = `SELECT ` + analysisColumns + ` FROM analyses WHERE practice_id = $1`

	a, err := scanAnalysis(conn(ctx, r.pool).QueryRow(ctx, q, practiceID.String()))
	if err != nil {
		return nil, mapError("analysis", err)
	}
	return a, nil
}

func scanAnalysis(row pgx.Row) (*analysis.Analysis, error) {
	var (
		id, practiceID, model, modelVersion, status string
		fragments                                   []byte
		createdAt                                   time.Time
	)
	if err := row.Scan(&id, &practiceID, &fragments, &model, &modelVersion, &status, &createdAt); err != nil {
		return nil, err
	}

	parsedID, err := domain.ParseID(id)
	if err != nil {
		return nil, fmt.Errorf("decode analysis id: %w", err)
	}
	parsedPracticeID, err := domain.ParseID(practiceID)
	if err != nil {
		return nil, fmt.Errorf("decode analysis practice id: %w", err)
	}

	var parsedFragments []analysis.Fragment
	if err := json.Unmarshal(fragments, &parsedFragments); err != nil {
		return nil, fmt.Errorf("decode analysis fragments: %w", err)
	}

	return &analysis.Analysis{
		ID:           parsedID,
		PracticeID:   parsedPracticeID,
		Fragments:    parsedFragments,
		Model:        model,
		ModelVersion: modelVersion,
		Status:       analysis.AnalysisStatus(status),
		CreatedAt:    createdAt,
	}, nil
}
