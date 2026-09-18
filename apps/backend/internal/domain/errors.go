package domain

import "fmt"

// ValidationError signals invalid input to a domain operation (HTTP 422).
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// InvalidStateError signals an operation that is not allowed in the current
// state of an aggregate (HTTP 409, wire code invalid_state).
type InvalidStateError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *InvalidStateError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NotFoundError signals that a referenced aggregate does not exist (HTTP 404,
// wire code not_found).
type NotFoundError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// AnalysisPendingError signals that an analysis result was requested before the
// analysis finished (HTTP 409, wire code analysis_pending). It is an idempotent
// success for POST /practices/{id}/analyze (AP4).
type AnalysisPendingError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *AnalysisPendingError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// AnalysisFailedError signals that an analysis finished with a failure. It has
// no dedicated 4xx status; the client observes it via analysis.status == failed
// (wire code analysis_failed, reserved).
type AnalysisFailedError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *AnalysisFailedError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// LLMUnavailableError signals that the LLM provider is down or timed out
// (HTTP 503, wire code llm_unavailable).
type LLMUnavailableError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *LLMUnavailableError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// InternalError signals an unexpected infrastructure failure (A5, HTTP 500).
// Adapters wrap unknown database or dependency errors in this type so no
// infrastructure detail leaks to the service.
type InternalError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *InternalError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
