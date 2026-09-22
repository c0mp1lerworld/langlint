// Package event_handlers holds the API entry-point subscribers of domain events
// (manifest §5.1). They are application services: they may call other services,
// repositories and the UnitOfWork, and must be idempotent.
package event_handlers

import "github.com/c0mp1lerworld/langlint/backend/internal/domain"

// analysisRequested accepts both value and pointer forms: services append value
// events, while the outbox relay rebuilds pointer events through NewEvent.
func analysisRequested(event domain.DomainEvent) (domain.AnalysisRequested, bool) {
	switch e := event.(type) {
	case domain.AnalysisRequested:
		return e, true
	case *domain.AnalysisRequested:
		return *e, true
	default:
		return domain.AnalysisRequested{}, false
	}
}

// analysisCompleted accepts both value and pointer forms.
func analysisCompleted(event domain.DomainEvent) (domain.AnalysisCompleted, bool) {
	switch e := event.(type) {
	case domain.AnalysisCompleted:
		return e, true
	case *domain.AnalysisCompleted:
		return *e, true
	default:
		return domain.AnalysisCompleted{}, false
	}
}

// analysisFailed accepts both value and pointer forms.
func analysisFailed(event domain.DomainEvent) (domain.AnalysisFailed, bool) {
	switch e := event.(type) {
	case domain.AnalysisFailed:
		return e, true
	case *domain.AnalysisFailed:
		return *e, true
	default:
		return domain.AnalysisFailed{}, false
	}
}

// weaknessDetected accepts both value and pointer forms.
func weaknessDetected(event domain.DomainEvent) (domain.WeaknessDetected, bool) {
	switch e := event.(type) {
	case domain.WeaknessDetected:
		return e, true
	case *domain.WeaknessDetected:
		return *e, true
	default:
		return domain.WeaknessDetected{}, false
	}
}
