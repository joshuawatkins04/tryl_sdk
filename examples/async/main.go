// Example demonstrates async usage of the Activity Logger SDK.
package main

import (
	"context"
	"log"
	"os"
	"sync"
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
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	var wg sync.WaitGroup

	events := []tryl.Event{
		{UserID: "user_1", Action: "page.viewed"},
		{UserID: "user_2", Action: "button.clicked"},
		{UserID: "user_3", Action: "form.submitted"},
	}

	for i, event := range events {
		wg.Add(1)
		resultCh := client.LogAsync(ctx, event)

		go func(idx int, ch <-chan tryl.AsyncResult) {
			defer wg.Done()

			result := <-ch
			if result.Error != nil {
				log.Printf("Event %d failed: %v", idx, result.Error)
			} else {
				log.Printf("Event %d logged: ID=%s", idx, result.Response.ID)
			}
		}(i, resultCh)
	}

	wg.Wait()

	log.Println("All async events processed")

	resultCh := client.LogAsync(ctx, tryl.Event{
		UserID: "user_fire_and_forget",
		Action: "notification.sent",
	})

	select {
	case result := <-resultCh:
		if result.Error != nil {
			log.Printf("Fire-and-forget failed (but we checked): %v", result.Error)
		} else {
			log.Printf("Fire-and-forget succeeded: %s", result.Response.ID)
		}
	case <-time.After(5 * time.Second):
		log.Println("Fire-and-forget timed out")
	}
}
