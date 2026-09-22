package repositories

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

// PostgresStudySessionRepository implements storage.StudySessionRepository.
type PostgresStudySessionRepository struct {
	pool *pgxpool.Pool
}

var _ storage.StudySessionRepository = (*PostgresStudySessionRepository)(nil)

// NewStudySessionRepository builds a repository over the given pool.
func NewStudySessionRepository(pool *pgxpool.Pool) *PostgresStudySessionRepository {
	return &PostgresStudySessionRepository{pool: pool}
}

const studySessionColumns = `id, user_id, profile, theory, traps, exercises, status, created_at`

// Save inserts the study session, matching the aggregate's identity by id.
func (r *PostgresStudySessionRepository) Save(ctx context.Context, s *tutor.StudySession) error {
	profile, err := json.Marshal(s.Profile.Entries)
	if err != nil {
		return &domain.InternalError{Field: "profile", Message: "cannot encode profile"}
	}
	traps, err := json.Marshal(s.Traps)
	if err != nil {
		return &domain.InternalError{Field: "traps", Message: "cannot encode traps"}
	}
	exercises, err := json.Marshal(s.Exercises)
	if err != nil {
		return &domain.InternalError{Field: "exercises", Message: "cannot encode exercises"}
	}

	const q = `
INSERT INTO study_sessions (` + studySessionColumns + `)
VALUES ($1, $2, $3::jsonb, $4, $5::jsonb, $6::jsonb, $7, $8)
ON CONFLICT (id) DO UPDATE SET
	user_id = EXCLUDED.user_id,
	profile = EXCLUDED.profile,
	theory = EXCLUDED.theory,
	traps = EXCLUDED.traps,
	exercises = EXCLUDED.exercises,
	status = EXCLUDED.status`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		s.ID.String(),
		s.UserID.String(),
		string(profile),
		s.Theory,
		string(traps),
		string(exercises),
		s.Status.String(),
		s.CreatedAt,
	); err != nil {
		return mapError("study_session", err)
	}
	return nil
}
