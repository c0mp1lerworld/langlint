package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

// StudySessionRepository persists the tutor.StudySession aggregate (A2). The
// tutor bounded context stores generated study sessions as owned data (A9), so
// it is keyed by the raw portable user id.
type StudySessionRepository interface {
	Save(ctx context.Context, session *tutor.StudySession) error
}
