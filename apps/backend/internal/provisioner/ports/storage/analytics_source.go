package storage

import (
	"context"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

// CompletedAnalysis is a completed analysis joined with its practice owner. It
// is the append-only source of truth that refresh-aggregates reconciles from
// (A4): analytics are derived data, never authoritative.
type CompletedAnalysis struct {
	UserID      domain.ID
	Fragments   []analysis.Fragment
	CompletedAt time.Time
}

// AnalyticsSourceRepository reads the source of truth of the analytics bounded
// context.
type AnalyticsSourceRepository interface {
	ListCompletedAnalyses(ctx context.Context) ([]CompletedAnalysis, error)
}
