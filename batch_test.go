package tryl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestBatcher_ResultMapping tests the critical bug where batch result mapping
// uses UserID+Action matching instead of index-based mapping.
// This test MUST FAIL before the bug fix and PASS after.
func TestBatcher_ResultMapping(t *testing.T) {
	t.Parallel()

	// Create events with DUPLICATE UserID+Action combinations
	// This is the scenario that triggers the bug
	events := []Event{
		{UserID: "user_1", Action: "user.created"}, // Index 0
		{UserID: "user_2", Action: "user.created"}, // Index 1
		{UserID: "user_1", Action: "user.created"}, // Index 2 - DUPLICATE of index 0!
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns results in order matching the request
		w.WriteHeader(http.StatusMultiStatus)
		resp := batchResponse{
			Results: []EventResponse{
				{ID: "evt_result_0", Timestamp: time.Now()}, // For index 0
				{ID: "evt_result_1", Timestamp: time.Now()}, // For index 1
				{ID: "evt_result_2", Timestamp: time.Now()}, // For index 2
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.LogBatch(context.Background(), events)
	if err != nil {
		t.Fatalf("LogBatch() error = %v", err)
	}

	// Verify each event gets the correct result by INDEX, not by matching fields
	if len(resp.Results) < 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}
	if resp.Results[0].ID != "evt_result_0" {
		t.Errorf("index 0: got %v, want evt_result_0", resp.Results[0].ID)
	}
	if resp.Results[1].ID != "evt_result_1" {
		t.Errorf("index 1: got %v, want evt_result_1", resp.Results[1].ID)
	}

	// THIS IS THE CRITICAL TEST: Index 2 should get evt_result_2
	// With the bug, it gets evt_result_0 because it matches user_1+user.created
	if resp.Results[2].ID != "evt_result_2" {
		t.Errorf("index 2: got %v, want evt_result_2 (BUG: result mapping by UserID+Action instead of index)", resp.Results[2].ID)
	}

	// Verify no errors in batch response
	if len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
			t.Errorf("unexpected error at index %d: %s", e.Index, e.Message)
		}
	}
}

func TestBatcher_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid event",
			event: Event{
				UserID: "user_123",
				Action: "user.created",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusMultiStatus)
				w.Write([]byte(`{"results":[{"id":"evt_123","timestamp":"2026-01-30T10:00:00Z"}]}`))
			}))
			defer server.Close()

			// Create client with batching enabled
			batchCfg := BatchConfig{
				MaxBatchSize:  10,
				FlushInterval: 100 * time.Millisecond,
			}
			client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef",
				WithBaseURL(server.URL),
				WithBatching(batchCfg))
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			defer client.Close()

			// Log async (uses batcher)
			resultCh := client.LogAsync(context.Background(), tt.event)

			if !tt.wantErr {
				// Wait for result
				select {
				case result := <-resultCh:
					if result.Error != nil {
						t.Errorf("unexpected error in result: %v", result.Error)
					}
					if result.Response == nil {
						t.Error("expected response, got nil")
					}
				case <-time.After(500 * time.Millisecond):
					t.Error("timeout waiting for batch result")
				}
			}
		})
	}
}

func TestBatcher_Flush(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(`{"results":[{"id":"evt_1","timestamp":"2026-01-30T10:00:00Z"}]}`))
	}))
	defer server.Close()

	batchCfg := BatchConfig{
		MaxBatchSize:  10,
		FlushInterval: 5 * time.Second, // Long interval, we'll flush manually
	}
	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef",
		WithBaseURL(server.URL),
		WithBatching(batchCfg))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Add one event
	_ = client.LogAsync(context.Background(), Event{
		UserID: "user_123",
		Action: "user.created",
	})

	// Manual flush
	if err := client.Flush(context.Background()); err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	// Verify server was called
	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}
}

func TestBatcher_Stop(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(`{"results":[{"id":"evt_123","timestamp":"2026-01-30T10:00:00Z"}]}`))
	}))
	defer server.Close()

	batchCfg := BatchConfig{
		MaxBatchSize:  10,
		FlushInterval: 100 * time.Millisecond,
	}
	client, err := NewClient("actlog_test_1234567890abcdef1234567890abcdef",
		WithBaseURL(server.URL),
		WithBatching(batchCfg))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Add event
	resultCh := client.LogAsync(context.Background(), Event{
		UserID: "user_123",
		Action: "user.created",
	})

	// Close client (triggers graceful shutdown)
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Pending event should still get result
	select {
	case result := <-resultCh:
		if result.Error != nil {
			t.Errorf("unexpected error after close: %v", result.Error)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for pending event result after close")
	}
}
