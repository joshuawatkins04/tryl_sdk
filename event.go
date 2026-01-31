package tryl

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event represents an activity event to be logged.
type Event struct {
	// UserID is the user who performed or is associated with the action. Required.
	UserID string `json:"user_id"`
	// Action is the type of action performed (e.g., "user.created"). Required.
	// Must be lowercase alphanumeric with dots or underscores.
	Action string `json:"action"`
	// ActorID is who performed the action (may differ from UserID). Optional.
	ActorID string `json:"actor_id,omitempty"`
	// TargetType is the type of resource affected (e.g., "document"). Optional.
	TargetType string `json:"target_type,omitempty"`
	// TargetID is the identifier of the affected resource. Optional.
	TargetID string `json:"target_id,omitempty"`
	// Metadata is additional structured data about the event. Optional.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Getter methods for validation interface compatibility.
func (e *Event) GetUserID() string          { return e.UserID }
func (e *Event) GetAction() string          { return e.Action }
func (e *Event) GetActorID() string         { return e.ActorID }
func (e *Event) GetTargetType() string      { return e.TargetType }
func (e *Event) GetTargetID() string        { return e.TargetID }
func (e *Event) GetMetadata() json.RawMessage { return e.Metadata }

// WithMetadata is a helper to set metadata from a map.
//
// Deprecated: This method silently ignores JSON marshaling errors.
// Use WithMetadataValidated instead, which returns validation errors.
// This method will be removed in v1.0.0.
func (e Event) WithMetadata(m map[string]any) Event {
	data, _ := json.Marshal(m)
	e.Metadata = data
	return e
}

// WithMetadataValidated sets metadata from a map with error handling.
// It returns an error if the metadata cannot be marshaled to JSON.
// This is the preferred method for setting metadata.
//
// Example:
//
//	event := tryl.Event{
//	    UserID: "user_123",
//	    Action: "document.created",
//	}
//	event, err := event.WithMetadataValidated(map[string]any{
//	    "title": "My Document",
//	    "size":  1024,
//	})
//	if err != nil {
//	    return fmt.Errorf("invalid metadata: %w", err)
//	}
func (e Event) WithMetadataValidated(m map[string]any) (Event, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return e, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	e.Metadata = data
	return e, nil
}

// SetMetadata sets metadata directly from json.RawMessage.
// This is useful when you already have validated JSON.
func (e Event) SetMetadata(metadata json.RawMessage) Event {
	e.Metadata = metadata
	return e
}

// EventResponse represents the API response after creating an event.
type EventResponse struct {
	// ID is the unique identifier for the created event.
	ID string `json:"id"`
	// Timestamp is when the event was recorded.
	Timestamp time.Time `json:"timestamp"`
}

// EventFilter represents query parameters for listing events.
type EventFilter struct {
	// UserID filters events by user.
	UserID string
	// ActorID filters events by actor.
	ActorID string
	// Action filters events by action type.
	// Supports wildcards: "org.*" matches "org.created", "org.updated", etc.
	// "*.created" matches "user.created", "org.created", etc.
	Action string

	// TargetType filters events by target resource type.
	TargetType string
	// TargetID filters events by target resource ID.
	TargetID string

	// StartTime filters events occurring at or after this time (inclusive).
	// Use nil to not filter by start time.
	StartTime *time.Time
	// EndTime filters events occurring at or before this time (inclusive).
	// Use nil to not filter by end time.
	EndTime *time.Time

	// MetadataContains filters events where metadata contains the specified JSON object.
	// Uses JSONB containment (@> operator in PostgreSQL).
	// Example: {"status": "active"} matches events with metadata containing this key-value.
	MetadataContains map[string]any
	// MetadataSearch performs full-text search in metadata.
	// Searches across all text fields in the metadata JSON.
	MetadataSearch string

	// Cursor is an opaque pagination cursor returned by the previous query.
	// When set, Offset is ignored (cursor-based pagination takes precedence).
	// Cursor-based pagination is more efficient for large result sets.
	Cursor string
	// Offset is the number of events to skip (offset-based pagination).
	// Deprecated: Use Cursor for better performance with large datasets.
	Offset int

	// Limit is the maximum number of events to return (max 100).
	Limit int
	// Order specifies the sort order: "asc" (oldest first) or "desc" (newest first).
	// Defaults to "desc" if not specified.
	Order string
}

// EventList represents the response when listing events.
type EventList struct {
	// Events is the list of events matching the filter.
	Events []StoredEvent `json:"events"`
	// HasMore indicates if there are more events to fetch.
	HasMore bool `json:"has_more"`
	// Total is the total count of matching events.
	// Only populated with offset-based pagination. Omitted with cursor-based pagination.
	Total int `json:"total,omitempty"`
	// NextCursor is the cursor to use for fetching the next page.
	// Only populated with cursor-based pagination when HasMore is true.
	NextCursor string `json:"next_cursor,omitempty"`
}

// StoredEvent represents an event retrieved from the API.
type StoredEvent struct {
	// ID is the unique identifier for the event.
	ID string `json:"id"`
	// UserID is the user associated with the event.
	UserID string `json:"user_id"`
	// Action is the type of action performed.
	Action string `json:"action"`
	// ActorID is who performed the action.
	ActorID string `json:"actor_id,omitempty"`
	// TargetType is the type of resource affected.
	TargetType string `json:"target_type,omitempty"`
	// TargetID is the identifier of the affected resource.
	TargetID string `json:"target_id,omitempty"`
	// Metadata is additional structured data about the event.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Timestamp is when the event was recorded.
	Timestamp time.Time `json:"timestamp"`
}

// batchRequest is the internal request format for batch operations.
type batchRequest struct {
	Events []Event `json:"events"`
}

// batchResponse is the internal response format for batch operations.
type batchResponse struct {
	Results []EventResponse    `json:"results"`
	Errors  []batchResultError `json:"errors"`
}

// batchResultError represents an error for a specific event in a batch.
type batchResultError struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
