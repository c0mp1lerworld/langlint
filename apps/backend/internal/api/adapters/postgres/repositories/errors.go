package repositories

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// mapError converts infrastructure errors into domain errors (A5), so no pgx
// detail ever reaches the service.
func mapError(field string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.NotFoundError{Field: field, Message: "not found"}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ValidationError{Field: field, Message: "already exists"}
	}

	return &domain.InternalError{Field: field, Message: "database error"}
}
