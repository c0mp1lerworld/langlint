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
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// PostgresPracticeRepository implements storage.PracticeRepository.
type PostgresPracticeRepository struct {
	pool *pgxpool.Pool
}

var _ storage.PracticeRepository = (*PostgresPracticeRepository)(nil)

// NewPracticeRepository builds a repository over the given pool.
func NewPracticeRepository(pool *pgxpool.Pool) *PostgresPracticeRepository {
	return &PostgresPracticeRepository{pool: pool}
}

const practiceColumns = `id, user_id, source_text, draft_text, target_rules, status, created_at, updated_at, deleted_at`

// Save upserts the practice, matching the aggregate's identity by id.
func (r *PostgresPracticeRepository) Save(ctx context.Context, p *practice.Practice) error {
	rules, err := json.Marshal(p.TargetRules)
	if err != nil {
		return &domain.InternalError{Field: "target_rules", Message: "cannot encode target rules"}
	}

	const q = `
INSERT INTO practices (` + practiceColumns + `)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
	user_id = EXCLUDED.user_id,
	source_text = EXCLUDED.source_text,
	draft_text = EXCLUDED.draft_text,
	target_rules = EXCLUDED.target_rules,
	status = EXCLUDED.status,
	updated_at = EXCLUDED.updated_at,
	deleted_at = EXCLUDED.deleted_at`

	if _, err := conn(ctx, r.pool).Exec(ctx, q,
		p.ID.String(),
		p.UserID.String(),
		p.SourceText.String(),
		p.DraftText.String(),
		string(rules),
		p.Status.String(),
		p.CreatedAt,
		p.UpdatedAt,
		p.DeletedAt,
	); err != nil {
		return mapError("practice", err)
	}
	return nil
}

// GetByID returns a non-deleted practice by id.
func (r *PostgresPracticeRepository) GetByID(ctx context.Context, id domain.ID) (*practice.Practice, error) {
	const q = `SELECT ` + practiceColumns + ` FROM practices WHERE id = $1 AND deleted_at IS NULL`

	p, err := scanPractice(conn(ctx, r.pool).QueryRow(ctx, q, id.String()))
	if err != nil {
		return nil, mapError("practice", err)
	}
	return p, nil
}

// ListByUser returns a page of the user's non-deleted practices plus the total.
func (r *PostgresPracticeRepository) ListByUser(ctx context.Context, userID domain.ID, limit, offset int) ([]practice.Practice, int, error) {
	q := conn(ctx, r.pool)

	var total int
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM practices WHERE user_id = $1 AND deleted_at IS NULL`,
		userID.String(),
	).Scan(&total); err != nil {
		return nil, 0, mapError("practice", err)
	}

	rows, err := q.Query(ctx,
		`SELECT `+practiceColumns+` FROM practices
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC, id
		 LIMIT $2 OFFSET $3`,
		userID.String(), limit, offset,
	)
	if err != nil {
		return nil, 0, mapError("practice", err)
	}
	defer rows.Close()

	practices := make([]practice.Practice, 0)
	for rows.Next() {
		p, err := scanPractice(rows)
		if err != nil {
			return nil, 0, mapError("practice", err)
		}
		practices = append(practices, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapError("practice", err)
	}

	return practices, total, nil
}

// ListAllByUser returns every non-deleted practice of the user, unpaginated, for
// the A9 data export.
func (r *PostgresPracticeRepository) ListAllByUser(ctx context.Context, userID domain.ID) ([]practice.Practice, error) {
	rows, err := conn(ctx, r.pool).Query(ctx,
		`SELECT `+practiceColumns+` FROM practices
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at, id`,
		userID.String(),
	)
	if err != nil {
		return nil, mapError("practice", err)
	}
	defer rows.Close()

	practices := make([]practice.Practice, 0)
	for rows.Next() {
		p, err := scanPractice(rows)
		if err != nil {
			return nil, mapError("practice", err)
		}
		practices = append(practices, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("practice", err)
	}
	return practices, nil
}

func scanPractice(row pgx.Row) (*practice.Practice, error) {
	var (
		id, userID, sourceText, draftText, status string
		rules                                     []byte
		createdAt, updatedAt                      time.Time
		deletedAt                                 *time.Time
	)
	if err := row.Scan(&id, &userID, &sourceText, &draftText, &rules, &status, &createdAt, &updatedAt, &deletedAt); err != nil {
		return nil, err
	}

	parsedID, err := domain.ParseID(id)
	if err != nil {
		return nil, fmt.Errorf("decode practice id: %w", err)
	}
	parsedUserID, err := domain.ParseID(userID)
	if err != nil {
		return nil, fmt.Errorf("decode practice user id: %w", err)
	}

	var targetRules []practice.TargetRule
	if err := json.Unmarshal(rules, &targetRules); err != nil {
		return nil, fmt.Errorf("decode practice target rules: %w", err)
	}

	return &practice.Practice{
		ID:          parsedID,
		UserID:      parsedUserID,
		SourceText:  practice.SourceText(sourceText),
		DraftText:   practice.DraftText(draftText),
		TargetRules: targetRules,
		Status:      practice.PracticeStatus(status),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   deletedAt,
	}, nil
}
