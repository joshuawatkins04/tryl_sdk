// Example demonstrates batching with the Activity Logger SDK.
package main

import (
	"context"
	"fmt"
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
		tryl.WithBatching(tryl.BatchConfig{
			MaxBatchSize:  10,
			FlushInterval: 2 * time.Second,
			OnError: func(events []tryl.Event, err error) {
				log.Printf("Batch error: %d events failed: %v", len(events), err)
			},
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	log.Println("Sending 25 events with batching (batch size: 10)...")

	for i := 0; i < 25; i++ {
		wg.Add(1)
		resultCh := client.LogAsync(ctx, tryl.Event{
			UserID: fmt.Sprintf("user_%d", i),
			Action: "batch.test",
		})

		go func(idx int, ch <-chan tryl.AsyncResult) {
			defer wg.Done()

			result := <-ch
			mu.Lock()
			defer mu.Unlock()

			if result.Error != nil {
				failCount++
				log.Printf("Event %d failed: %v", idx, result.Error)
			} else {
				successCount++
			}
		}(i, resultCh)
	}

	log.Println("All events queued, waiting for results...")

	log.Println("Flushing remaining events...")
	if err := client.Flush(ctx); err != nil {
		log.Printf("Flush error: %v", err)
	}

	wg.Wait()

	log.Printf("Results: %d succeeded, %d failed", successCount, failCount)

	if err := client.Close(); err != nil {
		log.Printf("Close error: %v", err)
	}

	log.Println("Done!")
}
