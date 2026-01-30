package tryl

import (
	"context"
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
			apiKey:  "actlog_test_valid",
			opts:    []Option{WithBaseURL("https://custom.example.com")},
			wantErr: false,
		},
		{
			name:    "with timeout option",
			apiKey:  "actlog_test_valid",
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

	client, err := NewClient("actlog_test_key", WithBaseURL(server.URL))
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

	client, err := NewClient("actlog_test_key", WithBaseURL(server.URL))
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

	client, err := NewClient("actlog_test_key", WithBaseURL(server.URL))
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

	client, err := NewClient("actlog_test_key", WithBaseURL(server.URL))
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
