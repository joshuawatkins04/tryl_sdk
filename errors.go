package tryl

import (
	"errors"
	"fmt"
)

// Error codes returned by the API.
const (
	ErrCodeInvalidRequest  = "invalid_request"
	ErrCodeValidationError = "validation_error"
	ErrCodeUnauthorized    = "unauthorized"
	ErrCodeForbidden       = "forbidden"
	ErrCodeNotFound        = "not_found"
	ErrCodeRateLimited     = "rate_limited"
	ErrCodeInternalError   = "internal_error"
)

// Sentinel errors for common conditions.
var (
	// ErrUnauthorized indicates invalid or missing API key.
	ErrUnauthorized = errors.New("tryl: unauthorized")

	// ErrRateLimited indicates the request was rate limited.
	ErrRateLimited = errors.New("tryl: rate limited")

	// ErrValidation indicates a validation error in the request.
	ErrValidation = errors.New("tryl: validation error")
)

// APIError represents an error response from the Activity Logger API.
type APIError struct {
	// HTTPStatus is the HTTP status code.
	HTTPStatus int
	// Code is the error code from the API.
	Code string
	// Message is the human-readable error message.
	Message string
	// RequestID is the unique identifier for the request (for support).
	RequestID string
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("tryl: %s (code=%s, status=%d, request_id=%s)",
			e.Message, e.Code, e.HTTPStatus, e.RequestID)
	}
	return fmt.Sprintf("tryl: %s (code=%s, status=%d)",
		e.Message, e.Code, e.HTTPStatus)
}

// Is implements errors.Is support for sentinel errors.
func (e *APIError) Is(target error) bool {
	switch {
	case target == ErrUnauthorized:
		return e.HTTPStatus == 401
	case target == ErrRateLimited:
		return e.HTTPStatus == 429
	case target == ErrValidation:
		return e.Code == ErrCodeValidationError
	default:
		return false
	}
}

// IsRetryable returns true if the error is potentially retryable.
func (e *APIError) IsRetryable() bool {
	return e.HTTPStatus >= 500 || e.HTTPStatus == 429
}

// IsUnauthorized reports whether the error is an authorization error.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatus == 401
	}
	return errors.Is(err, ErrUnauthorized)
}

// IsRateLimited reports whether the error indicates rate limiting.
func IsRateLimited(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatus == 429
	}
	return errors.Is(err, ErrRateLimited)
}

// IsValidationError reports whether the error is a validation error.
func IsValidationError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == ErrCodeValidationError
	}
	return errors.Is(err, ErrValidation)
}

// NetworkError wraps network-related errors.
type NetworkError struct {
	Op  string // Operation that failed (e.g., "dial", "read")
	Err error  // Underlying error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("tryl: network error during %s: %v", e.Op, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// IsTemporary reports whether the error is temporary and may succeed on retry.
func (e *NetworkError) IsTemporary() bool {
	var temp interface{ Temporary() bool }
	if errors.As(e.Err, &temp) {
		return temp.Temporary()
	}
	return true
}
