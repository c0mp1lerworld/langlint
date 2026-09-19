package services

import (
	"context"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/storage"
)

// PurgeRawDataService enforces the limited retention of raw practice data (A8):
// practices that were soft-deleted longer than the retention window ago are
// physically removed.
type PurgeRawDataService struct {
	raw       storage.RawDataRepository
	retention time.Duration
	now       func() time.Time
}

// NewPurgeRawDataService wires the job through its port. retention is the time
// a soft-deleted practice is kept before being purged.
func NewPurgeRawDataService(raw storage.RawDataRepository, retention time.Duration) *PurgeRawDataService {
	return &PurgeRawDataService{raw: raw, retention: retention, now: time.Now}
}

// Purge removes the practices whose deletion happened before the retention
// cutoff. It returns how many practices were purged. It is idempotent.
func (s *PurgeRawDataService) Purge(ctx context.Context) (int, error) {
	cutoff := s.now().Add(-s.retention)
	return s.raw.PurgePracticesDeletedBefore(ctx, cutoff)
}
