package outreach

import (
	"errors"
	"fmt"
)

// ErrorCode represents a typed service-level error category.
type ErrorCode string

const (
	ErrorValidation   ErrorCode = "validation"
	ErrorNotFound     ErrorCode = "not_found"
	ErrorConflict     ErrorCode = "conflict"
	ErrorUnauthorized ErrorCode = "unauthorized"
	ErrorForbidden    ErrorCode = "forbidden"
	ErrorInternal     ErrorCode = "internal"
)

// ServiceError wraps a service error with a stable code for HTTP translation.
type ServiceError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *ServiceError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

func newServiceError(code ErrorCode, msg string, cause error) error {
	return &ServiceError{Code: code, Message: msg, Cause: cause}
}

func IsErrorCode(err error, code ErrorCode) bool {
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		return false
	}
	return serviceErr.Code == code
}
