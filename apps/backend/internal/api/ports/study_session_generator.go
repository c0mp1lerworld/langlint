package ports

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

// StudySessionRequest is the aggregated weakness context used to generate a
// study session (checklist 8.3). It carries only aggregated data — error codes,
// severities and frequencies — never raw learner text, so no PII ever leaves the
// process (A8, PRODUCT_DOMAIN §12.1).
type StudySessionRequest struct {
	Profile tutor.WeaknessProfile
}

// StudySessionContent is the LLM-generated content of a session: the summarized
// theory, the learner's common traps and the interactive exercises (checklist
// 8.3.2). It excludes the session identity (id, user, timestamps) so the caller
// can assemble the tutor.StudySession aggregate.
type StudySessionContent struct {
	Theory    string
	Traps     []tutor.Trap
	Exercises []tutor.Exercise
}

// StudySessionGenerator is the port of the study-session engine (checklist
// 8.3.1): it generates the session content from a weakness profile using
// Structured Outputs. Like LLMExtractor and TutorQuestioner, it is
// provider-agnostic (A1) and runs outside any transaction.
type StudySessionGenerator interface {
	Generate(ctx context.Context, req StudySessionRequest) (StudySessionContent, error)
}
