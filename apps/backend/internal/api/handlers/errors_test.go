package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestDomainError_MapsDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   ErrorResponseCode
	}{
		{"validation", &domain.ValidationError{Message: "bad"}, http.StatusUnprocessableEntity, ErrorResponseCodeValidationError},
		{"invalid state", &domain.InvalidStateError{Message: "bad state"}, http.StatusConflict, ErrorResponseCodeInvalidState},
		{"not found", &domain.NotFoundError{Message: "missing"}, http.StatusNotFound, ErrorResponseCodeNotFound},
		{"analysis pending", &domain.AnalysisPendingError{Message: "pending"}, http.StatusConflict, ErrorResponseCodeAnalysisPending},
		{"analysis failed", &domain.AnalysisFailedError{Message: "failed"}, http.StatusUnprocessableEntity, ErrorResponseCodeAnalysisFailed},
		{"llm unavailable", &domain.LLMUnavailableError{Message: "down"}, http.StatusServiceUnavailable, ErrorResponseCodeLlmUnavailable},
		{"internal", &domain.InternalError{Message: "boom"}, http.StatusInternalServerError, ErrorResponseCodeInternal},
		{"unknown", errors.New("raw pgx error"), http.StatusInternalServerError, ErrorResponseCodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, _ := domainError(tt.err)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if code != tt.wantCode {
				t.Fatalf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestWriteDomainError_UnknownError_DoesNotLeakDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	writeDomainError(rec, errors.New(`pq: duplicate key value violates unique constraint "practices_pkey"`))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Code != ErrorResponseCodeInternal {
		t.Fatalf("code = %q, want %q", body.Code, ErrorResponseCodeInternal)
	}
	if body.Message != "internal error" {
		t.Fatalf("message = %q, want generic", body.Message)
	}
}

func TestWriteNotImplemented_Returns501(t *testing.T) {
	rec := httptest.NewRecorder()
	writeNotImplemented(rec)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body.Code != ErrorResponseCodeNotImplemented {
		t.Fatalf("code = %q, want %q", body.Code, ErrorResponseCodeNotImplemented)
	}
}
