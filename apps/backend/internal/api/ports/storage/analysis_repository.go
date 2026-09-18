package storage

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// AnalysisRepository persists the Analysis aggregate (A2).
type AnalysisRepository interface {
	Save(ctx context.Context, analysis *analysis.Analysis) error
	GetByPracticeID(ctx context.Context, practiceID domain.ID) (*analysis.Analysis, error)
}
