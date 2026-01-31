package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// actionRegexp matches the server-side validation.
// Source: internal/models/event.go:116
// CRITICAL: Keep in sync with server validation.
var actionRegexp = regexp.MustCompile(`^[a-z][a-z0-9_.]*[a-z0-9]$`)

const maxFieldLength = 255

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string
	Message string
	Value   string
}

func (e *FieldError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("%s: %s (got: %s)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// EventValidator defines the interface for event validation.
// This allows validation to work with both Event and future event types
// without code duplication.
type EventValidator interface {
	GetUserID() string
	GetAction() string
	GetActorID() string
	GetTargetType() string
	GetTargetID() string
	GetMetadata() json.RawMessage
}

// ValidateEvent validates an event according to server-side rules.
// Server validation source: internal/models/event.go:129-168
//
// Returns nil if valid, or a FieldError describing the first validation failure.
func ValidateEvent(e EventValidator) error {
	// UserID validation (required)
	if e.GetUserID() == "" {
		return &FieldError{Field: "user_id", Message: "is required"}
	}
	if len(e.GetUserID()) > maxFieldLength {
		return &FieldError{
			Field:   "user_id",
			Message: fmt.Sprintf("must be %d characters or less", maxFieldLength),
			Value:   truncateForDisplay(e.GetUserID()),
		}
	}

	// Action validation (required)
	if e.GetAction() == "" {
		return &FieldError{Field: "action", Message: "is required"}
	}
	if len(e.GetAction()) > maxFieldLength {
		return &FieldError{
			Field:   "action",
			Message: fmt.Sprintf("must be %d characters or less", maxFieldLength),
			Value:   e.GetAction(),
		}
	}
	if !actionRegexp.MatchString(e.GetAction()) {
		return &FieldError{
			Field:   "action",
			Message: "must be lowercase alphanumeric with dots or underscores (e.g., 'user.created', 'org_member_added')",
			Value:   e.GetAction(),
		}
	}

	// Optional field validations
	if e.GetActorID() != "" && len(e.GetActorID()) > maxFieldLength {
		return &FieldError{
			Field:   "actor_id",
			Message: fmt.Sprintf("must be %d characters or less", maxFieldLength),
			Value:   truncateForDisplay(e.GetActorID()),
		}
	}

	if e.GetTargetType() != "" && len(e.GetTargetType()) > maxFieldLength {
		return &FieldError{
			Field:   "target_type",
			Message: fmt.Sprintf("must be %d characters or less", maxFieldLength),
			Value:   truncateForDisplay(e.GetTargetType()),
		}
	}

	if e.GetTargetID() != "" && len(e.GetTargetID()) > maxFieldLength {
		return &FieldError{
			Field:   "target_id",
			Message: fmt.Sprintf("must be %d characters or less", maxFieldLength),
			Value:   truncateForDisplay(e.GetTargetID()),
		}
	}

	// Metadata validation (must be valid JSON if present)
	if len(e.GetMetadata()) > 0 {
		var js json.RawMessage
		if err := json.Unmarshal(e.GetMetadata(), &js); err != nil {
			return &FieldError{
				Field:   "metadata",
				Message: fmt.Sprintf("must be valid JSON: %v", err),
			}
		}
	}

	return nil
}

// ValidateAction validates just the action field format.
// Useful for pre-validation before constructing an Event.
func ValidateAction(action string) error {
	if action == "" {
		return &FieldError{Field: "action", Message: "is required"}
	}
	if len(action) > maxFieldLength {
		return &FieldError{
			Field:   "action",
			Message: fmt.Sprintf("must be %d characters or less", maxFieldLength),
			Value:   action,
		}
	}
	if !actionRegexp.MatchString(action) {
		return &FieldError{
			Field:   "action",
			Message: "must be lowercase alphanumeric with dots or underscores",
			Value:   action,
		}
	}
	return nil
}

// truncateForDisplay truncates a string to 50 chars for error display.
func truncateForDisplay(s string) string {
	if len(s) <= 50 {
		return s
	}
	return s[:50] + "..."
}
