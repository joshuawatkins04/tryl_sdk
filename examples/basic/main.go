// Example demonstrates basic usage of the Activity Logger SDK.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joshuawatkins04/tryl_sdk"
)

func main() {
	apiKey := os.Getenv("ACTIVITY_LOGGER_API_KEY")
	if apiKey == "" {
		log.Fatal("ACTIVITY_LOGGER_API_KEY environment variable is required")
	}

	baseURL := os.Getenv("ACTIVITY_LOGGER_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	client, err := tryl.NewClient(
		apiKey,
		tryl.WithBaseURL(baseURL),
		tryl.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	metadata, _ := json.Marshal(map[string]any{
		"document_title": "Q1 Report",
		"file_size":      1024,
	})
	resp, err := client.Log(ctx, tryl.Event{
		UserID:     "user_123",
		Action:     "document.created",
		ActorID:    "user_456",
		TargetType: "document",
		TargetID:   "doc_789",
		Metadata:   metadata,
	})
	if err != nil {
		if tryl.IsValidationError(err) {
			log.Printf("Validation error: %v", err)
		} else if tryl.IsUnauthorized(err) {
			log.Printf("Unauthorized: %v", err)
		} else {
			log.Printf("Failed to log event: %v", err)
		}
		return
	}

	log.Printf("Event logged successfully: ID=%s, Timestamp=%s", resp.ID, resp.Timestamp)

	list, err := client.List(ctx, tryl.EventFilter{
		UserID: "user_123",
		Limit:  10,
	})
	if err != nil {
		log.Printf("Failed to list events: %v", err)
		return
	}

	log.Printf("Found %d events (total: %d, has_more: %v)", len(list.Events), list.Total, list.HasMore)
	for _, event := range list.Events {
		log.Printf("  - %s: %s at %s", event.ID, event.Action, event.Timestamp)
	}
}
