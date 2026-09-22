package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// PostgresQuizAttemptRepository implements storage.QuizAttemptRepository.
type PostgresQuizAttemptRepository struct {
	pool *pgxpool.Pool
}

var _ storage.QuizAttemptRepository = (*PostgresQuizAttemptRepository)(nil)

// NewQuizAttemptRepository builds a repository over the given pool.
func NewQuizAttemptRepository(pool *pgxpool.Pool) *PostgresQuizAttemptRepository {
	return &PostgresQuizAttemptRepository{pool: pool}
}

const quizAttemptColumns = `id, user_id, practice_id, correct, created_at`

// Append inserts a quiz attempt. It is append-only: no upsert, no update.
func (r *PostgresQuizAttemptRepository) Append(ctx context.Context, attempt *analytics.QuizAttempt) error {
	const q = `
INSERT INTO quiz_attempts (` + quizAttemptColumns + `)
VALUES ($1, $2, $3, $4, $5)`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		attempt.ID.String(),
		attempt.UserID.String(),
		attempt.PracticeID.String(),
		attempt.Correct,
		attempt.CreatedAt,
	); err != nil {
		return mapError("quiz_attempt", err)
	}
	return nil
}

// ListByUser returns every quiz attempt of the user, newest first.
func (r *PostgresQuizAttemptRepository) ListByUser(ctx context.Context, userID domain.ID) ([]analytics.QuizAttempt, error) {
	rows, err := conn(ctx, r.pool).Query(ctx,
		`SELECT `+quizAttemptColumns+` FROM quiz_attempts
		 WHERE user_id = $1
		 ORDER BY created_at DESC, id`,
		userID.String(),
	)
	if err != nil {
		return nil, mapError("quiz_attempt", err)
	}
	defer rows.Close()

	attempts := make([]analytics.QuizAttempt, 0)
	for rows.Next() {
		attempt, err := scanQuizAttempt(rows)
		if err != nil {
			return nil, mapError("quiz_attempt", err)
		}
		attempts = append(attempts, *attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("quiz_attempt", err)
	}
	return attempts, nil
}

func scanQuizAttempt(row pgx.Row) (*analytics.QuizAttempt, error) {
	var (
		id, userID, practiceID string
		correct                bool
		createdAt              time.Time
	)
	if err := row.Scan(&id, &userID, &practiceID, &correct, &createdAt); err != nil {
		return nil, err
	}

	parsedID, err := domain.ParseID(id)
	if err != nil {
		return nil, fmt.Errorf("decode quiz attempt id: %w", err)
	}
	parsedUserID, err := domain.ParseID(userID)
	if err != nil {
		return nil, fmt.Errorf("decode quiz attempt user id: %w", err)
	}
	parsedPracticeID, err := domain.ParseID(practiceID)
	if err != nil {
		return nil, fmt.Errorf("decode quiz attempt practice id: %w", err)
	}

	return &analytics.QuizAttempt{
		ID:         parsedID,
		UserID:     parsedUserID,
		PracticeID: parsedPracticeID,
		Correct:    correct,
		CreatedAt:  createdAt,
	}, nil
}
