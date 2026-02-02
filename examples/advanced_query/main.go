package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joshuawatkins04/tryl_sdk"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ACTLOG_API_KEY")
	if apiKey == "" {
		log.Fatal("ACTLOG_API_KEY environment variable is required")
	}

	// Create client
	client, err := tryl.NewClient(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Basic query with user filter
	fmt.Println("=== Example 1: Filter by UserID ===")
	result1, err := client.List(ctx, tryl.EventFilter{
		UserID: "user_123",
		Limit:  10,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d events for user_123\n", len(result1.Events))
	fmt.Printf("Has more: %v\n\n", result1.HasMore)

	// Example 2: Query with wildcard action filter
	fmt.Println("=== Example 2: Wildcard Action Filter ===")
	result2, err := client.List(ctx, tryl.EventFilter{
		Action: "user.*", // Matches "user.created", "user.updated", etc.
		Limit:  10,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d events with action 'user.*'\n", len(result2.Events))
	for _, event := range result2.Events {
		fmt.Printf("  - %s: %s (user: %s)\n", event.ID, event.Action, event.UserID)
	}
	fmt.Println()

	// Example 3: Time range query (last 24 hours)
	fmt.Println("=== Example 3: Time Range Filter ===")
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	result3, err := client.List(ctx, tryl.EventFilter{
		StartTime: &yesterday,
		EndTime:   &now,
		Limit:     10,
		Order:     "desc", // Newest first
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d events in the last 24 hours\n", len(result3.Events))
	for _, event := range result3.Events {
		fmt.Printf("  - %s at %s\n", event.Action, event.Timestamp.Format(time.RFC3339))
	}
	fmt.Println()

	// Example 4: Target filters (specific resource)
	fmt.Println("=== Example 4: Target Filters ===")
	result4, err := client.List(ctx, tryl.EventFilter{
		TargetType: "document",
		TargetID:   "doc_789",
		Limit:      10,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d events for document doc_789\n", len(result4.Events))
	fmt.Println()

	// Example 5: Metadata containment query (JSONB)
	fmt.Println("=== Example 5: Metadata Contains Filter ===")
	result5, err := client.List(ctx, tryl.EventFilter{
		MetadataContains: map[string]any{
			"status": "active",
		},
		Limit: 10,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d events with status=active in metadata\n", len(result5.Events))
	for _, event := range result5.Events {
		var metadata map[string]any
		if err := json.Unmarshal(event.Metadata, &metadata); err == nil {
			fmt.Printf("  - %s: %v\n", event.Action, metadata)
		}
	}
	fmt.Println()

	// Example 6: Metadata full-text search
	fmt.Println("=== Example 6: Metadata Search ===")
	result6, err := client.List(ctx, tryl.EventFilter{
		MetadataSearch: "important",
		Limit:          10,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d events with 'important' in metadata\n", len(result6.Events))
	fmt.Println()

	// Example 7: Cursor-based pagination (efficient for large datasets)
	fmt.Println("=== Example 7: Cursor-Based Pagination ===")
	filter := tryl.EventFilter{
		Limit: 5, // Small limit to demonstrate pagination
		Order: "desc",
	}

	page := 1
	for {
		result, err := client.List(ctx, filter)
		if err != nil {
			log.Fatalf("Failed to list events: %v", err)
		}

		fmt.Printf("Page %d: %d events\n", page, len(result.Events))
		for _, event := range result.Events {
			fmt.Printf("  - %s: %s\n", event.ID, event.Action)
		}

		if !result.HasMore {
			fmt.Println("No more results")
			break
		}

		// Use NextCursor for next page
		filter.Cursor = result.NextCursor
		page++

		if page > 3 {
			fmt.Println("(Stopping after 3 pages for demo)")
			break
		}
	}
	fmt.Println()

	// Example 8: Complex multi-filter query
	fmt.Println("=== Example 8: Complex Multi-Filter Query ===")
	oneWeekAgo := now.Add(-7 * 24 * time.Hour)
	result8, err := client.List(ctx, tryl.EventFilter{
		Action:     "document.*",
		TargetType: "document",
		StartTime:  &oneWeekAgo,
		EndTime:    &now,
		MetadataContains: map[string]any{
			"priority": "high",
		},
		Limit: 10,
		Order: "desc",
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("Found %d high-priority document events in the last week\n", len(result8.Events))
	fmt.Println()

	fmt.Println("All examples completed successfully!")
}
