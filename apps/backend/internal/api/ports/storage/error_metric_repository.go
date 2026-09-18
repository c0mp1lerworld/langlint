package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

// ErrorMetricRepository materializes error-pattern metrics keyed by
// (UserID, Code, Window) (PRODUCT_DOMAIN §7.1).
type ErrorMetricRepository interface {
	Upsert(ctx context.Context, metric *analytics.ErrorMetric) error
	ListByUser(ctx context.Context, userID domain.ID, window analytics.Window) ([]analytics.ErrorMetric, error)
}
