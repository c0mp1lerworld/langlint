package storage

import (
	"context"
	"time"
)

// RawDataRepository purges raw practice data past its retention window (A8).
type RawDataRepository interface {
	// PurgePracticesDeletedBefore physically removes every soft-deleted
	// practice whose deletion happened before cutoff, together with its
	// analyses. It returns how many practices were purged.
	PurgePracticesDeletedBefore(ctx context.Context, cutoff time.Time) (int, error)
}
