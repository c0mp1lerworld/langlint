package services

import (
	"context"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
)

// ExecuteDeletionsService materializes the right to be forgotten (A9) once the
// grace period elapsed. Until then the request stays pending, so a user can
// still change their mind.
type ExecuteDeletionsService struct {
	deletions storage.DeletionRepository
	grace     time.Duration
	now       func() time.Time
}

// NewExecuteDeletionsService wires the job through its port. grace is the time
// a deletion request waits before it is executed.
func NewExecuteDeletionsService(deletions storage.DeletionRepository, grace time.Duration) *ExecuteDeletionsService {
	return &ExecuteDeletionsService{deletions: deletions, grace: grace, now: time.Now}
}

// Run executes every pending request whose grace period elapsed and returns how
// many were executed. It is idempotent: only pending requests are listed, and
// the aggregate refuses a double execution.
func (s *ExecuteDeletionsService) Run(ctx context.Context) (int, error) {
	now := s.now()

	pending, err := s.deletions.ListPending(ctx)
	if err != nil {
		return 0, err
	}

	executed := 0
	for i := range pending {
		req := pending[i]
		if !req.Due(now, s.grace) {
			continue
		}
		if err := req.MarkExecuted(now); err != nil {
			return executed, err
		}
		if err := s.deletions.Execute(ctx, &req); err != nil {
			return executed, err
		}
		executed++
	}
	return executed, nil
}
