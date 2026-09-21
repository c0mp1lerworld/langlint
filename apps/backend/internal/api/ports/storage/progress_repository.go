package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// ProgressRepository reads the raw per-analysis samples that feed the progress
// series. Progress is derived on read from the append-only analyses (A4); it is
// not materialized, so there is no drift to reconcile.
type ProgressRepository interface {
	ListSamplesByUser(ctx context.Context, userID domain.ID) ([]analytics.ProgressSample, error)
}
