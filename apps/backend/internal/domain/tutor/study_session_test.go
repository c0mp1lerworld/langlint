package tutor_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func TestStudySession_NewStudySession_Valid_StartsGenerated(t *testing.T) {
	session, err := tutor.NewStudySession(domain.MustNewID(), validProfile(t), "Resumida teoría.", validTraps(t), validExercises(t), testNow)
	if err != nil {
		t.Fatalf("NewStudySession() error = %v", err)
	}
	if session.ID.IsZero() {
		t.Fatal("ID is zero, want a generated UUID v7")
	}
	if session.Status != tutor.StudySessionStatusGenerated {
		t.Fatalf("Status = %q, want %q", session.Status, tutor.StudySessionStatusGenerated)
	}
	if session.Theory != "Resumida teoría." {
		t.Fatalf("Theory = %q, want trimmed", session.Theory)
	}
	if len(session.Traps) != 1 {
		t.Fatalf("Traps len = %d, want 1", len(session.Traps))
	}
	if len(session.Exercises) != 1 {
		t.Fatalf("Exercises len = %d, want 1", len(session.Exercises))
	}
	if !session.CreatedAt.Equal(testNow) {
		t.Fatalf("CreatedAt = %v, want %v", session.CreatedAt, testNow)
	}
}

func TestStudySession_NewStudySession_EmptyProfile_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewStudySession(domain.MustNewID(), tutor.WeaknessProfile{}, "theory", validTraps(t), validExercises(t), testNow)
	assertValidationError(t, err, "profile")
}

func TestStudySession_NewStudySession_EmptyTheory_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewStudySession(domain.MustNewID(), validProfile(t), "   ", validTraps(t), validExercises(t), testNow)
	assertValidationError(t, err, "theory")
}

func TestStudySession_NewStudySession_EmptyTraps_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewStudySession(domain.MustNewID(), validProfile(t), "theory", nil, validExercises(t), testNow)
	assertValidationError(t, err, "traps")
}

func TestStudySession_NewStudySession_EmptyExercises_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewStudySession(domain.MustNewID(), validProfile(t), "theory", validTraps(t), nil, testNow)
	assertValidationError(t, err, "exercises")
}

func TestStudySession_Start_FromGenerated_SetsActive(t *testing.T) {
	session := validSession(t)

	if err := session.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if session.Status != tutor.StudySessionStatusActive {
		t.Fatalf("Status = %q, want %q", session.Status, tutor.StudySessionStatusActive)
	}
}

func TestStudySession_Start_NotGenerated_ReturnsInvalidState(t *testing.T) {
	session := validSession(t)
	if err := session.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err := session.Start()
	assertInvalidStateError(t, err, "status")
}

func TestStudySession_Complete_FromActive_SetsCompleted(t *testing.T) {
	session := validSession(t)
	if err := session.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := session.Complete(); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if session.Status != tutor.StudySessionStatusCompleted {
		t.Fatalf("Status = %q, want %q", session.Status, tutor.StudySessionStatusCompleted)
	}
}

func TestStudySession_Complete_NotActive_ReturnsInvalidState(t *testing.T) {
	session := validSession(t)

	err := session.Complete()
	assertInvalidStateError(t, err, "status")
}

func TestStudySession_MarshalJSON_UsesSnakeCase(t *testing.T) {
	session := validSession(t)

	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "user_id", "profile", "theory", "traps", "exercises", "status", "created_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
}

func validSession(t *testing.T) *tutor.StudySession {
	t.Helper()
	session, err := tutor.NewStudySession(domain.MustNewID(), validProfile(t), "Resumida teoría.", validTraps(t), validExercises(t), testNow)
	if err != nil {
		t.Fatalf("NewStudySession() error = %v", err)
	}
	return session
}

func validProfile(t *testing.T) tutor.WeaknessProfile {
	t.Helper()
	entry, err := tutor.NewWeaknessEntry(domain.ErrorPatternCodePrepositionInfinitive, domain.ErrorPatternSeverityModerate, 4, testNow)
	if err != nil {
		t.Fatalf("NewWeaknessEntry() error = %v", err)
	}
	profile, err := tutor.NewWeaknessProfile([]tutor.WeaknessEntry{entry})
	if err != nil {
		t.Fatalf("NewWeaknessProfile() error = %v", err)
	}
	return profile
}

func validTraps(t *testing.T) []tutor.Trap {
	t.Helper()
	trap, err := tutor.NewTrap(domain.ErrorPatternCodePrepositionInfinitive, "Using \"to\" before a gerund")
	if err != nil {
		t.Fatalf("NewTrap() error = %v", err)
	}
	return []tutor.Trap{trap}
}

func validExercises(t *testing.T) []tutor.Exercise {
	t.Helper()
	exercise, err := tutor.NewExercise(tutor.ExerciseKindFill, "I look forward ___ (see) you.", "to seeing")
	if err != nil {
		t.Fatalf("NewExercise() error = %v", err)
	}
	return []tutor.Exercise{exercise}
}

func assertInvalidStateError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var target *domain.InvalidStateError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.InvalidStateError", err)
	}
	if target.Field != field {
		t.Fatalf("InvalidStateError.Field = %q, want %q", target.Field, field)
	}
}
