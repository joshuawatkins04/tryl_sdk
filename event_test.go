package tryl

import (
	"encoding/json"
	"testing"
)

func TestEvent_WithMetadataValidated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]any
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: map[string]any{
				"key": "value",
				"num": 123,
			},
			wantErr: false,
		},
		{
			name:     "empty metadata",
			metadata: map[string]any{},
			wantErr:  false,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			wantErr:  false,
		},
		{
			name: "nested metadata",
			metadata: map[string]any{
				"user": map[string]any{
					"email": "test@example.com",
					"age":   25,
				},
			},
			wantErr: false,
		},
		{
			name: "unmarshalable type - channel",
			metadata: map[string]any{
				"channel": make(chan int),
			},
			wantErr: true,
		},
		{
			name: "unmarshalable type - function",
			metadata: map[string]any{
				"func": func() {},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := Event{
				UserID: "user_123",
				Action: "test.action",
			}

			newEvent, err := event.WithMetadataValidated(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithMetadataValidated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.metadata != nil {
				// Verify metadata was set correctly
				var result map[string]any
				if err := json.Unmarshal(newEvent.Metadata, &result); err != nil {
					t.Errorf("failed to unmarshal result metadata: %v", err)
				}
			}
		})
	}
}

func TestEvent_WithMetadata_Deprecated(t *testing.T) {
	t.Parallel()

	// Test that deprecated method still works for valid input
	event := Event{
		UserID: "user_123",
		Action: "test.action",
	}

	metadata := map[string]any{
		"key": "value",
	}

	newEvent := event.WithMetadata(metadata)

	var result map[string]any
	if err := json.Unmarshal(newEvent.Metadata, &result); err != nil {
		t.Errorf("deprecated WithMetadata() failed to set valid metadata: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("metadata key = %v, want %v", result["key"], "value")
	}
}

func TestEvent_SetMetadata(t *testing.T) {
	t.Parallel()

	event := Event{
		UserID: "user_123",
		Action: "test.action",
	}

	rawJSON := json.RawMessage(`{"existing":"json"}`)
	newEvent := event.SetMetadata(rawJSON)

	if string(newEvent.Metadata) != string(rawJSON) {
		t.Errorf("SetMetadata() = %s, want %s", newEvent.Metadata, rawJSON)
	}
}

func TestEvent_GetterMethods(t *testing.T) {
	t.Parallel()

	event := &Event{
		UserID:     "user_123",
		Action:     "test.action",
		ActorID:    "actor_456",
		TargetType: "document",
		TargetID:   "doc_789",
		Metadata:   json.RawMessage(`{"key":"value"}`),
	}

	if event.GetUserID() != "user_123" {
		t.Errorf("GetUserID() = %v, want user_123", event.GetUserID())
	}
	if event.GetAction() != "test.action" {
		t.Errorf("GetAction() = %v, want test.action", event.GetAction())
	}
	if event.GetActorID() != "actor_456" {
		t.Errorf("GetActorID() = %v, want actor_456", event.GetActorID())
	}
	if event.GetTargetType() != "document" {
		t.Errorf("GetTargetType() = %v, want document", event.GetTargetType())
	}
	if event.GetTargetID() != "doc_789" {
		t.Errorf("GetTargetID() = %v, want doc_789", event.GetTargetID())
	}
	if string(event.GetMetadata()) != `{"key":"value"}` {
		t.Errorf("GetMetadata() = %v, want {\"key\":\"value\"}", string(event.GetMetadata()))
	}
}
