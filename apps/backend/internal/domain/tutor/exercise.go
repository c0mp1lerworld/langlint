package tutor

import (
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// ExerciseKind is the type of interactive exercise a session contains
// (PRODUCT_DOMAIN §12.1). Only production-oriented kinds are supported, mirroring
// the quiz's open/fill kinds.
type ExerciseKind string

// Known exercise kinds.
const (
	ExerciseKindOpen ExerciseKind = "open"
	ExerciseKindFill ExerciseKind = "fill"
)

// IsValid reports whether the kind is one of the known values.
func (k ExerciseKind) IsValid() bool {
	switch k {
	case ExerciseKindOpen, ExerciseKindFill:
		return true
	default:
		return false
	}
}

// Exercise is an interactive practice item in a study session (PRODUCT_DOMAIN
// §12.1, checklist 8.1.2: "ejercicios interactivos").
type Exercise struct {
	Kind   ExerciseKind `json:"kind"`
	Prompt string       `json:"prompt"`
	Answer string       `json:"answer"`
}

// NewExercise validates kind, prompt and answer and returns an Exercise value
// object with its text fields trimmed.
func NewExercise(kind ExerciseKind, prompt, answer string) (Exercise, error) {
	if !kind.IsValid() {
		return Exercise{}, &domain.ValidationError{Field: "exercise.kind", Message: "must be open or fill"}
	}
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return Exercise{}, &domain.ValidationError{Field: "exercise.prompt", Message: "must not be empty"}
	}
	trimmedAnswer := strings.TrimSpace(answer)
	if trimmedAnswer == "" {
		return Exercise{}, &domain.ValidationError{Field: "exercise.answer", Message: "must not be empty"}
	}
	return Exercise{Kind: kind, Prompt: trimmedPrompt, Answer: trimmedAnswer}, nil
}
