package tryl

import (
	"errors"
	"fmt"
)

// Error codes returned by the API.
const (
	ErrCodeInvalidRequest   = "invalid_request"
	ErrCodeValidationError  = "validation_error"
	ErrCodeUnauthorized     = "unauthorized"
	ErrCodeForbidden        = "forbidden"
	ErrCodeNotFound         = "not_found"
	ErrCodeProjectNotFound  = "project_not_found"
	ErrCodeKeyNotFound      = "key_not_found"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeInternalError    = "internal_error"
)

// Sentinel errors for common conditions.
var (
	// ErrUnauthorized indicates invalid or missing API key.
	ErrUnauthorized = errors.New("tryl: unauthorized")

	// ErrRateLimited indicates the request was rate limited.
	ErrRateLimited = errors.New("tryl: rate limited")

	// ErrValidation indicates a validation error in the request.
	ErrValidation = errors.New("tryl: validation error")

	// ErrInvalidAPIKey indicates the API key format is invalid.
	ErrInvalidAPIKey = errors.New("tryl: invalid API key format")

	// ErrProjectNotFound indicates the requested project was not found.
	ErrProjectNotFound = errors.New("tryl: project not found")

	// ErrKeyNotFound indicates the requested API key was not found.
	ErrKeyNotFound = errors.New("tryl: API key not found")
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
	case target == ErrProjectNotFound:
		return e.Code == ErrCodeProjectNotFound || (e.HTTPStatus == 404 && e.Code == ErrCodeNotFound)
	case target == ErrKeyNotFound:
		return e.Code == ErrCodeKeyNotFound || (e.HTTPStatus == 404 && e.Code == ErrCodeNotFound)
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

// ValidationError represents a client-side validation error.
// This wraps validation failures from the internal validation package
// and provides a consistent public error type.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string
	// Message is the human-readable error message.
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("tryl: validation error: %s: %s", e.Field, e.Message)
}

// Is implements errors.Is support.
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// IsClientValidationError reports whether the error is a client-side validation error.
// This distinguishes client-side validation errors from server-side validation errors.
func IsClientValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
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

// IsProjectNotFound reports whether the error indicates a project was not found.
func IsProjectNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == ErrCodeProjectNotFound || (apiErr.HTTPStatus == 404 && apiErr.Code == ErrCodeNotFound)
	}
	return errors.Is(err, ErrProjectNotFound)
}

// IsKeyNotFound reports whether the error indicates an API key was not found.
func IsKeyNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == ErrCodeKeyNotFound || (apiErr.HTTPStatus == 404 && apiErr.Code == ErrCodeNotFound)
	}
	return errors.Is(err, ErrKeyNotFound)
}
