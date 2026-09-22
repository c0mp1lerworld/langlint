package tutor_test

import (
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func TestExerciseKind_IsValid_KnownAndUnknownValues(t *testing.T) {
	cases := []struct {
		kind tutor.ExerciseKind
		want bool
	}{
		{tutor.ExerciseKindOpen, true},
		{tutor.ExerciseKindFill, true},
		{tutor.ExerciseKind("mcq"), false},
		{tutor.ExerciseKind(""), false},
	}
	for _, c := range cases {
		if got := c.kind.IsValid(); got != c.want {
			t.Fatalf("%q.IsValid() = %v, want %v", c.kind, got, c.want)
		}
	}
}

func TestExercise_NewExercise_Valid_TrimsPromptAndAnswer(t *testing.T) {
	exercise, err := tutor.NewExercise(tutor.ExerciseKindFill, "  I look forward ___ (see) you.  ", "  to seeing  ")
	if err != nil {
		t.Fatalf("NewExercise() error = %v", err)
	}
	if exercise.Kind != tutor.ExerciseKindFill {
		t.Fatalf("Kind = %q, want %q", exercise.Kind, tutor.ExerciseKindFill)
	}
	if exercise.Prompt != "I look forward ___ (see) you." {
		t.Fatalf("Prompt = %q, want trimmed", exercise.Prompt)
	}
	if exercise.Answer != "to seeing" {
		t.Fatalf("Answer = %q, want trimmed", exercise.Answer)
	}
}

func TestExercise_NewExercise_UnknownKind_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewExercise("mcq", "prompt", "answer")
	assertValidationError(t, err, "exercise.kind")
}

func TestExercise_NewExercise_EmptyPrompt_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewExercise(tutor.ExerciseKindOpen, "   ", "answer")
	assertValidationError(t, err, "exercise.prompt")
}

func TestExercise_NewExercise_EmptyAnswer_ReturnsValidationError(t *testing.T) {
	_, err := tutor.NewExercise(tutor.ExerciseKindOpen, "prompt", "   ")
	assertValidationError(t, err, "exercise.answer")
}
