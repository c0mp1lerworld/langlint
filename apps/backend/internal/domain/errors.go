package domain

import "fmt"

// ValidationError signals invalid input to a domain operation (HTTP 422).
type ValidationError struct {
	Field   string
	Message string
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
	Field   string
	Message string
}

// Error implements the error interface.
func (e *InvalidStateError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
