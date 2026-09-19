package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// ErrorMetricRepository is the provisioner view of the materialized error
// metrics. ReplaceAll reconciles them atomically from the source of truth, so a
// drifted table is rebuilt rather than incrementally patched (A4).
type ErrorMetricRepository interface {
	ReplaceAll(ctx context.Context, metrics []*analytics.ErrorMetric) error
}
