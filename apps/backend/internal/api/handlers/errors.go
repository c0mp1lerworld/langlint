package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// writeDomainError maps a domain error to its HTTP status and wire body (A5).
// Unexpected errors become a generic 500; no infrastructure detail is exposed.
func writeDomainError(w http.ResponseWriter, err error) {
	status, code, message := domainError(err)
	writeError(w, status, code, message)
}

// writeError encodes the standard ErrorResponse body.
func writeError(w http.ResponseWriter, status int, code ErrorResponseCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Code: code, Message: message})
}

// domainError translates a domain error to (status, wire code, message). The
// message always comes from the domain error, never from an infrastructure
// error (A5).
func domainError(err error) (int, ErrorResponseCode, string) {
	var validation *domain.ValidationError
	var invalidState *domain.InvalidStateError
	var notFound *domain.NotFoundError
	var pending *domain.AnalysisPendingError
	var failed *domain.AnalysisFailedError
	var llmUnavailable *domain.LLMUnavailableError
	var llmTruncated *domain.LLMOutputTruncatedError
	var internal *domain.InternalError

	switch {
	case errors.As(err, &validation):
		return http.StatusUnprocessableEntity, ErrorResponseCodeValidationError, validation.Message
	case errors.As(err, &invalidState):
		return http.StatusConflict, ErrorResponseCodeInvalidState, invalidState.Message
	case errors.As(err, &notFound):
		return http.StatusNotFound, ErrorResponseCodeNotFound, notFound.Message
	case errors.As(err, &pending):
		return http.StatusConflict, ErrorResponseCodeAnalysisPending, pending.Message
	case errors.As(err, &failed):
		return http.StatusUnprocessableEntity, ErrorResponseCodeAnalysisFailed, failed.Message
	case errors.As(err, &llmUnavailable):
		return http.StatusServiceUnavailable, ErrorResponseCodeLlmUnavailable, llmUnavailable.Message
	case errors.As(err, &llmTruncated):
		return http.StatusServiceUnavailable, ErrorResponseCodeLlmOutputTruncated, llmTruncated.Message
	case errors.As(err, &internal):
		return http.StatusInternalServerError, ErrorResponseCodeInternal, "internal error"
	default:
		return http.StatusInternalServerError, ErrorResponseCodeInternal, "internal error"
	}
}
