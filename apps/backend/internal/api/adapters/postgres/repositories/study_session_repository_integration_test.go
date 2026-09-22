//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func mustStudySession(t *testing.T, userID domain.ID, createdAt time.Time) *tutor.StudySession {
	t.Helper()
	profile, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{{
		Code:       domain.ErrorPatternCodeWordOrder,
		Severity:   domain.ErrorPatternSeverityModerate,
		Count:      6,
		LastSeenAt: createdAt,
	}})
	require.NoError(t, err)
	trap, err := tutor.NewTrap(domain.ErrorPatternCodeWordOrder, "Colocar el verbo al final.")
	require.NoError(t, err)
	exercise, err := tutor.NewExercise(tutor.ExerciseKindFill, "I ___ (never) have seen.", "have never")
	require.NoError(t, err)
	session, err := tutor.NewStudySession(userID, profile, "El orden en inglés es SVO.", []tutor.Trap{trap}, []tutor.Exercise{exercise}, createdAt)
	require.NoError(t, err)
	return session
}

func TestPostgresStudySessionRepository_Save_Persists(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewStudySessionRepository(pool)
	userID := domain.MustNewID()
	session := mustStudySession(t, userID, time.Now().UTC().Truncate(time.Microsecond))

	require.NoError(t, repo.Save(ctx, session))

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM study_sessions WHERE id = $1`, session.ID.String()).Scan(&count))
	require.Equal(t, 1, count)
}

func TestPostgresStudySessionRepository_Save_UpsertsById(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := repositories.NewStudySessionRepository(pool)
	userID := domain.MustNewID()
	session := mustStudySession(t, userID, time.Now().UTC().Truncate(time.Microsecond))

	require.NoError(t, repo.Save(ctx, session))
	require.NoError(t, session.Start())
	require.NoError(t, repo.Save(ctx, session))

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM study_sessions WHERE id = $1`, session.ID.String()).Scan(&status))
	require.Equal(t, string(tutor.StudySessionStatusActive), status)
}
