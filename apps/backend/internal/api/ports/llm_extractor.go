package ports

import (
	"context"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// ExtractRequest carries the anonymized input of a single extraction (A8). The
// LLM call runs outside the transaction (PRODUCT_DOMAIN §4.7).
type ExtractRequest struct {
	PracticeID  domain.ID
	SourceText  string
	DraftText   string
	TargetRules []practice.TargetRule
}

// LLMExtractor is the port over the LLM provider (PRODUCT_DOMAIN §6.1). The
// domain stays provider-agnostic (A1). Model and ModelVersion expose the
// provider traceability the Analysis aggregate persists (PRODUCT_DOMAIN §4.2.3).
type LLMExtractor interface {
	Extract(ctx context.Context, req ExtractRequest) ([]analysis.Fragment, error)
	Model() string
	ModelVersion() string
}
