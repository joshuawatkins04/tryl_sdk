package tryl

import (
	"encoding/json"
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

// WithMetadata is a helper to set metadata from a map.
func (e Event) WithMetadata(m map[string]any) Event {
	data, _ := json.Marshal(m)
	e.Metadata = data
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
	Action string
	// Limit is the maximum number of events to return (max 100).
	Limit int
	// Offset is the number of events to skip.
	Offset int
}

// EventList represents the response when listing events.
type EventList struct {
	// Events is the list of events matching the filter.
	Events []StoredEvent `json:"events"`
	// HasMore indicates if there are more events to fetch.
	HasMore bool `json:"has_more"`
	// Total is the total count of matching events.
	Total int `json:"total"`
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
