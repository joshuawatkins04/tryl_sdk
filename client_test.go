package tryl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		apiKey  string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "valid api key",
			apiKey:  "actlog_test_1234567890abcdef1234567890abcdef1234567890",
			wantErr: false,
		},
		{
			name:    "empty api key",
			apiKey:  "",
			wantErr: true,
		},
		{
			name:    "with custom base URL",
			apiKey:  "actlog_test_1234567890abcdef1234567890abcdef",
			opts:    []Option{WithBaseURL("https://custom.example.com")},
			wantErr: false,
		},
		{
			name:    "with timeout option",
			apiKey:  "actlog_test_1234567890abcdef1234567890abcdef",
			opts:    []Option{WithTimeout(5 * time.Second)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewClient(tt.apiKey, tt.opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestClient_Log_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("got method %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/v1/events") {
			t.Errorf("got path %s, want to contain /v1/events", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("missing Authorization header")
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"evt_abc123","timestamp":"2026-01-30T10:00:00Z"}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	event := Event{
		UserID: "user_123",
		Action: "user.created",
	}

	resp, err := client.Log(context.Background(), event)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	if resp.ID != "evt_abc123" {
		t.Errorf("got ID %q, want %q", resp.ID, "evt_abc123")
	}
}

func TestClient_Log_ValidationError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"validation_error","message":"user_id is required"}}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	event := Event{
		UserID: "",
		Action: "user.created",
	}

	_, err = client.Log(context.Background(), event)
	if err == nil {
		t.Error("expected error for validation failure, got nil")
	}
}

func TestClient_Log_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"evt_123","timestamp":"2026-01-30T10:00:00Z"}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := Event{
		UserID: "user_123",
		Action: "user.created",
	}

	_, err = client.Log(ctx, event)
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
}

func TestClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %s, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events":[{"id":"evt_1","user_id":"user_123","action":"user.created","timestamp":"2026-01-30T10:00:00Z"}],"has_more":false,"total":1}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.List(context.Background(), EventFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(resp.Events) != 1 {
		t.Errorf("got %d events, want 1", len(resp.Events))
	}

	if resp.Events[0].ID != "evt_1" {
		t.Errorf("got event ID %q, want %q", resp.Events[0].ID, "evt_1")
	}
}

func TestClient_List_EnhancedQueryFeatures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		filter         EventFilter
		wantQueryParam string
		wantValue      string
	}{
		{
			name: "target_type filter",
			filter: EventFilter{
				TargetType: "document",
			},
			wantQueryParam: "target_type",
			wantValue:      "document",
		},
		{
			name: "target_id filter",
			filter: EventFilter{
				TargetID: "doc_123",
			},
			wantQueryParam: "target_id",
			wantValue:      "doc_123",
		},
		{
			name: "metadata_search filter",
			filter: EventFilter{
				MetadataSearch: "important",
			},
			wantQueryParam: "metadata_search",
			wantValue:      "important",
		},
		{
			name: "order asc",
			filter: EventFilter{
				Order: "asc",
			},
			wantQueryParam: "order",
			wantValue:      "asc",
		},
		{
			name: "order desc",
			filter: EventFilter{
				Order: "desc",
			},
			wantQueryParam: "order",
			wantValue:      "desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the query parameter is sent correctly
				if got := r.URL.Query().Get(tt.wantQueryParam); got != tt.wantValue {
					t.Errorf("query param %s = %q, want %q", tt.wantQueryParam, got, tt.wantValue)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"events":[],"has_more":false,"total":0}`))
			}))
			defer server.Close()

			client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			_, err = client.List(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
		})
	}
}

func TestClient_List_TimeRangeFilters(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify time parameters are formatted correctly
		if got := r.URL.Query().Get("start_time"); got != startTime.Format(time.RFC3339) {
			t.Errorf("start_time = %q, want %q", got, startTime.Format(time.RFC3339))
		}
		if got := r.URL.Query().Get("end_time"); got != endTime.Format(time.RFC3339) {
			t.Errorf("end_time = %q, want %q", got, endTime.Format(time.RFC3339))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events":[],"has_more":false,"total":0}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	filter := EventFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	_, err = client.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestClient_List_MetadataContainsFilter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify metadata_contains is marshaled correctly
		metadataContains := r.URL.Query().Get("metadata_contains")
		if metadataContains == "" {
			t.Error("metadata_contains query param missing")
		}

		// Should be valid JSON
		var parsed map[string]any
		if err := json.Unmarshal([]byte(metadataContains), &parsed); err != nil {
			t.Errorf("metadata_contains is not valid JSON: %v", err)
		}

		if parsed["status"] != "active" {
			t.Errorf("metadata_contains status = %v, want active", parsed["status"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events":[],"has_more":false,"total":0}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	filter := EventFilter{
		MetadataContains: map[string]any{
			"status": "active",
		},
	}

	_, err = client.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestClient_List_CursorPaginationPrecedence(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify cursor is sent and offset is NOT sent when both are provided
		if got := r.URL.Query().Get("cursor"); got != "test_cursor_123" {
			t.Errorf("cursor = %q, want test_cursor_123", got)
		}
		if offset := r.URL.Query().Get("offset"); offset != "" {
			t.Errorf("offset should not be sent when cursor is present, got %q", offset)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events":[],"has_more":true,"next_cursor":"next_cursor_456"}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	filter := EventFilter{
		Cursor: "test_cursor_123",
		Offset: 100, // Should be ignored
	}

	resp, err := client.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Verify next_cursor is returned
	if resp.NextCursor != "next_cursor_456" {
		t.Errorf("NextCursor = %q, want next_cursor_456", resp.NextCursor)
	}
}

func TestClient_List_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events":[],"has_more":false,"total":10}`))
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Empty filter should still work (all new fields have safe zero values)
	resp, err := client.List(context.Background(), EventFilter{})
	if err != nil {
		t.Fatalf("List() with empty filter error = %v", err)
	}

	// Offset pagination should still work
	resp, err = client.List(context.Background(), EventFilter{
		UserID: "user_123",
		Limit:  10,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("List() with offset pagination error = %v", err)
	}

	if resp.Total != 10 {
		t.Errorf("Total = %d, want 10", resp.Total)
	}
}
