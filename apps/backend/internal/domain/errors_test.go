package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestValidationError_Error_IncludesFieldAndMessage(t *testing.T) {
	err := &domain.ValidationError{Field: "email", Message: "must be a valid email address"}

	msg := err.Error()
	if msg == "" {
		t.Fatal("Error() must not be empty")
	}
	for _, want := range []string{"email", "must be a valid email address"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Error() = %q, want it to contain %q", msg, want)
		}
	}
}

func TestValidationError_AsDomainError(t *testing.T) {
	var target *domain.ValidationError
	err := error(&domain.ValidationError{Field: "email", Message: "invalid"})

	if !errors.As(err, &target) {
		t.Fatal("errors.As must match *domain.ValidationError")
	}
}

func TestValidationError_Error_WithoutField_ReturnsMessage(t *testing.T) {
	err := &domain.ValidationError{Message: "invalid input"}

	if got := err.Error(); got != "invalid input" {
		t.Fatalf("Error() = %q, want %q", got, "invalid input")
	}
}

func TestInvalidStateError_Error_IncludesFieldAndMessage(t *testing.T) {
	err := &domain.InvalidStateError{Field: "status", Message: "must be draft to be edited"}

	msg := err.Error()
	for _, want := range []string{"status", "must be draft to be edited"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Error() = %q, want it to contain %q", msg, want)
		}
	}
}

func TestInvalidStateError_Error_WithoutField_ReturnsMessage(t *testing.T) {
	err := &domain.InvalidStateError{Message: "invalid state"}

	if got := err.Error(); got != "invalid state" {
		t.Fatalf("Error() = %q, want %q", got, "invalid state")
	}
}

func TestInvalidStateError_AsDomainError(t *testing.T) {
	var target *domain.InvalidStateError
	err := error(&domain.InvalidStateError{Field: "status", Message: "invalid"})

	if !errors.As(err, &target) {
		t.Fatal("errors.As must match *domain.InvalidStateError")
	}
}

func TestDomainErrors_Error_IncludeFieldAndMessage(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		field string
		msg   string
	}{
		{"NotFoundError", &domain.NotFoundError{Field: "practice", Message: "not found"}, "practice", "not found"},
		{"AnalysisPendingError", &domain.AnalysisPendingError{Field: "analysis", Message: "still pending"}, "analysis", "still pending"},
		{"AnalysisFailedError", &domain.AnalysisFailedError{Field: "analysis", Message: "has failed"}, "analysis", "has failed"},
		{"LLMUnavailableError", &domain.LLMUnavailableError{Field: "llm", Message: "unavailable"}, "llm", "unavailable"},
		{"InternalError", &domain.InternalError{Field: "db", Message: "unexpected"}, "db", "unexpected"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			for _, want := range []string{tc.field, tc.msg} {
				if !strings.Contains(got, want) {
					t.Fatalf("Error() = %q, want it to contain %q", got, want)
				}
			}
		})
	}
}

func TestDomainErrors_Error_WithoutField_ReturnsMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		msg  string
	}{
		{"NotFoundError", &domain.NotFoundError{Message: "not found"}, "not found"},
		{"AnalysisPendingError", &domain.AnalysisPendingError{Message: "still pending"}, "still pending"},
		{"AnalysisFailedError", &domain.AnalysisFailedError{Message: "has failed"}, "has failed"},
		{"LLMUnavailableError", &domain.LLMUnavailableError{Message: "unavailable"}, "unavailable"},
		{"InternalError", &domain.InternalError{Message: "unexpected"}, "unexpected"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.msg {
				t.Fatalf("Error() = %q, want %q", got, tc.msg)
			}
		})
	}
}

func TestDomainErrors_AsDomainError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		as   func(error) bool
	}{
		{"NotFoundError", &domain.NotFoundError{Message: "not found"}, func(e error) bool {
			var target *domain.NotFoundError
			return errors.As(e, &target)
		}},
		{"AnalysisPendingError", &domain.AnalysisPendingError{Message: "pending"}, func(e error) bool {
			var target *domain.AnalysisPendingError
			return errors.As(e, &target)
		}},
		{"AnalysisFailedError", &domain.AnalysisFailedError{Message: "failed"}, func(e error) bool {
			var target *domain.AnalysisFailedError
			return errors.As(e, &target)
		}},
		{"LLMUnavailableError", &domain.LLMUnavailableError{Message: "unavailable"}, func(e error) bool {
			var target *domain.LLMUnavailableError
			return errors.As(e, &target)
		}},
		{"InternalError", &domain.InternalError{Message: "unexpected"}, func(e error) bool {
			var target *domain.InternalError
			return errors.As(e, &target)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.as(tc.err) {
				t.Fatalf("errors.As must match %s", tc.name)
			}
		})
	}
}
