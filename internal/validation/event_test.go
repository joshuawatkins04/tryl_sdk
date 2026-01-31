package validation

import (
	"encoding/json"
	"strings"
	"testing"
)

// mockEvent implements the EventValidator interface for testing.
type mockEvent struct {
	UserID     string
	Action     string
	ActorID    string
	TargetType string
	TargetID   string
	Metadata   json.RawMessage
}

func (m *mockEvent) GetUserID() string          { return m.UserID }
func (m *mockEvent) GetAction() string          { return m.Action }
func (m *mockEvent) GetActorID() string         { return m.ActorID }
func (m *mockEvent) GetTargetType() string      { return m.TargetType }
func (m *mockEvent) GetTargetID() string        { return m.TargetID }
func (m *mockEvent) GetMetadata() json.RawMessage { return m.Metadata }

func TestValidateEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		event     *mockEvent
		wantErr   bool
		wantField string
	}{
		{
			name: "valid event minimal",
			event: &mockEvent{
				UserID: "user_123",
				Action: "user.created",
			},
			wantErr: false,
		},
		{
			name: "valid event all fields",
			event: &mockEvent{
				UserID:     "user_123",
				Action:     "document.updated",
				ActorID:    "admin_456",
				TargetType: "document",
				TargetID:   "doc_789",
				Metadata:   json.RawMessage(`{"key":"value"}`),
			},
			wantErr: false,
		},
		{
			name: "missing user_id",
			event: &mockEvent{
				Action: "user.created",
			},
			wantErr:   true,
			wantField: "user_id",
		},
		{
			name: "missing action",
			event: &mockEvent{
				UserID: "user_123",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "user_id too long",
			event: &mockEvent{
				UserID: strings.Repeat("a", 256),
				Action: "user.created",
			},
			wantErr:   true,
			wantField: "user_id",
		},
		{
			name: "action too long",
			event: &mockEvent{
				UserID: "user_123",
				Action: strings.Repeat("a", 256),
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "invalid action format - uppercase",
			event: &mockEvent{
				UserID: "user_123",
				Action: "User.Created",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "invalid action format - starts with number",
			event: &mockEvent{
				UserID: "user_123",
				Action: "1user.created",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "invalid action format - hyphen",
			event: &mockEvent{
				UserID: "user_123",
				Action: "user-created",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "invalid action format - space",
			event: &mockEvent{
				UserID: "user_123",
				Action: "user created",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "invalid action format - ends with dot",
			event: &mockEvent{
				UserID: "user_123",
				Action: "user.created.",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "invalid action format - ends with underscore",
			event: &mockEvent{
				UserID: "user_123",
				Action: "user_created_",
			},
			wantErr:   true,
			wantField: "action",
		},
		{
			name: "valid action with underscores",
			event: &mockEvent{
				UserID: "user_123",
				Action: "org_member_added",
			},
			wantErr: false,
		},
		{
			name: "valid action with dots",
			event: &mockEvent{
				UserID: "user_123",
				Action: "user.profile.updated",
			},
			wantErr: false,
		},
		{
			name: "valid action with numbers",
			event: &mockEvent{
				UserID: "user_123",
				Action: "event123.created",
			},
			wantErr: false,
		},
		{
			name: "actor_id too long",
			event: &mockEvent{
				UserID:  "user_123",
				Action:  "user.created",
				ActorID: strings.Repeat("a", 256),
			},
			wantErr:   true,
			wantField: "actor_id",
		},
		{
			name: "target_type too long",
			event: &mockEvent{
				UserID:     "user_123",
				Action:     "user.created",
				TargetType: strings.Repeat("a", 256),
			},
			wantErr:   true,
			wantField: "target_type",
		},
		{
			name: "target_id too long",
			event: &mockEvent{
				UserID:   "user_123",
				Action:   "user.created",
				TargetID: strings.Repeat("a", 256),
			},
			wantErr:   true,
			wantField: "target_id",
		},
		{
			name: "invalid metadata - not JSON",
			event: &mockEvent{
				UserID:   "user_123",
				Action:   "user.created",
				Metadata: json.RawMessage(`not json`),
			},
			wantErr:   true,
			wantField: "metadata",
		},
		{
			name: "invalid metadata - unclosed brace",
			event: &mockEvent{
				UserID:   "user_123",
				Action:   "user.created",
				Metadata: json.RawMessage(`{"key":"value"`),
			},
			wantErr:   true,
			wantField: "metadata",
		},
		{
			name: "valid metadata - empty object",
			event: &mockEvent{
				UserID:   "user_123",
				Action:   "user.created",
				Metadata: json.RawMessage(`{}`),
			},
			wantErr: false,
		},
		{
			name: "valid metadata - nested",
			event: &mockEvent{
				UserID:   "user_123",
				Action:   "user.created",
				Metadata: json.RawMessage(`{"user":{"email":"test@example.com","age":25}}`),
			},
			wantErr: false,
		},
		{
			name: "user_id exactly 255 chars",
			event: &mockEvent{
				UserID: strings.Repeat("a", 255),
				Action: "user.created",
			},
			wantErr: false,
		},
		{
			name: "action exactly 255 chars",
			event: &mockEvent{
				UserID: "user_123",
				Action: "a" + strings.Repeat("b", 253) + "c",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateEvent(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantField != "" {
				fieldErr, ok := err.(*FieldError)
				if !ok {
					t.Errorf("expected *FieldError, got %T", err)
					return
				}
				if fieldErr.Field != tt.wantField {
					t.Errorf("error field = %q, want %q", fieldErr.Field, tt.wantField)
				}
			}
		})
	}
}

func TestValidateAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		{
			name:    "valid action simple",
			action:  "user.created",
			wantErr: false,
		},
		{
			name:    "valid action with underscores",
			action:  "org_member_added",
			wantErr: false,
		},
		{
			name:    "valid action with numbers",
			action:  "event123.updated",
			wantErr: false,
		},
		{
			name:    "valid action complex",
			action:  "user.profile.settings.updated",
			wantErr: false,
		},
		{
			name:    "empty action",
			action:  "",
			wantErr: true,
		},
		{
			name:    "action too long",
			action:  strings.Repeat("a", 256),
			wantErr: true,
		},
		{
			name:    "action with uppercase",
			action:  "User.Created",
			wantErr: true,
		},
		{
			name:    "action with hyphen",
			action:  "user-created",
			wantErr: true,
		},
		{
			name:    "action with space",
			action:  "user created",
			wantErr: true,
		},
		{
			name:    "action starts with number",
			action:  "1user.created",
			wantErr: true,
		},
		{
			name:    "action starts with dot",
			action:  ".user.created",
			wantErr: true,
		},
		{
			name:    "action starts with underscore",
			action:  "_user_created",
			wantErr: true,
		},
		{
			name:    "action ends with dot",
			action:  "user.created.",
			wantErr: true,
		},
		{
			name:    "action ends with underscore",
			action:  "user_created_",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFieldError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *FieldError
		want string
	}{
		{
			name: "with value",
			err: &FieldError{
				Field:   "action",
				Message: "invalid format",
				Value:   "User.Created",
			},
			want: "action: invalid format (got: User.Created)",
		},
		{
			name: "without value",
			err: &FieldError{
				Field:   "user_id",
				Message: "is required",
			},
			want: "user_id: is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("FieldError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
