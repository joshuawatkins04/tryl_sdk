# Activity Logger Go SDK

The official Go SDK for the Activity Logger API. Track user activity events with type-safe validation, advanced querying, and project management capabilities.

[![Go Reference](https://pkg.go.dev/badge/github.com/wato/tryl_sdk.svg)](https://pkg.go.dev/github.com/wato/tryl_sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/wato/tryl_sdk)](https://goreportcard.com/report/github.com/wato/tryl_sdk)

## Features

- ✅ **Type-safe event logging** with client-side validation
- ✅ **Advanced querying** with filters, time ranges, and metadata search
- ✅ **Project & API key management** with session token authentication
- ✅ **Async logging** with automatic batching support
- ✅ **Automatic retries** with exponential backoff
- ✅ **Cursor-based pagination** for efficient large result sets
- ✅ **Context support** for cancellation and timeouts
- ✅ **Comprehensive error handling** with typed errors

## Installation

```bash
go get github.com/wato/tryl_sdk
```

## Quick Start

### Event Logging

```go
package main

import (
	"context"
	"log"

	"github.com/wato/tryl_sdk"
)

func main() {
	// Create client with API key
	client, err := tryl.NewClient("actlog_live_your_api_key_here")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Log an event
	event := tryl.Event{
		UserID: "user_123",
		Action: "user.login",
	}

	resp, err := client.Log(context.Background(), event)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Event logged: %s at %s", resp.ID, resp.Timestamp)
}
```

## Table of Contents

- [Event Logging](#event-logging)
  - [Basic Event](#basic-event)
  - [Event with Metadata](#event-with-metadata)
  - [Batch Logging](#batch-logging)
  - [Async Logging](#async-logging)
- [Querying Events](#querying-events)
  - [Basic Filters](#basic-filters)
  - [Time Range Queries](#time-range-queries)
  - [Metadata Queries](#metadata-queries)
  - [Pagination](#pagination)
- [Project Management](#project-management)
  - [Managing Projects](#managing-projects)
  - [Managing API Keys](#managing-api-keys)
- [Validation](#validation)
- [Error Handling](#error-handling)
- [Configuration](#configuration)
- [Examples](#examples)

## Event Logging

### Basic Event

```go
event := tryl.Event{
	UserID: "user_123",
	Action: "document.created",
}

resp, err := client.Log(ctx, event)
if err != nil {
	log.Fatal(err)
}
```

### Event with Metadata

Use `WithMetadataValidated()` for type-safe metadata handling:

```go
event := tryl.Event{
	UserID:     "user_123",
	Action:     "document.updated",
	ActorID:    "admin_456",
	TargetType: "document",
	TargetID:   "doc_789",
}

// Add metadata with validation
event, err := event.WithMetadataValidated(map[string]any{
	"title":    "My Document",
	"size":     1024,
	"tags":     []string{"important", "draft"},
	"modified": time.Now(),
})
if err != nil {
	log.Fatal(err)
}

resp, err := client.Log(ctx, event)
```

**Note:** `WithMetadata()` is deprecated. Use `WithMetadataValidated()` to catch JSON marshaling errors. See [Migration Guide](#migration-guide) for details.

### Batch Logging

Send up to 100 events in a single request:

```go
events := []tryl.Event{
	{UserID: "user_1", Action: "user.login"},
	{UserID: "user_2", Action: "user.login"},
	{UserID: "user_3", Action: "user.signup"},
}

resp, err := client.LogBatch(ctx, events)
if err != nil {
	log.Fatal(err)
}

// Check for partial failures
for _, e := range resp.Errors {
	log.Printf("Event %d failed: %s - %s", e.Index, e.Code, e.Message)
}
```

### Async Logging

Log events asynchronously with automatic batching:

```go
// Enable batching (groups events sent within 1 second)
client, err := tryl.NewClient(apiKey,
	tryl.WithBatching(1*time.Second, 50), // 1s interval, max 50 events
)

// Log asynchronously
resultCh := client.LogAsync(ctx, event)

// Process result when needed
result := <-resultCh
if result.Error != nil {
	log.Printf("Failed: %v", result.Error)
} else {
	log.Printf("Success: %s", result.Response.ID)
}

// Flush before shutdown
client.Flush(ctx)
```

## Querying Events

### Basic Filters

```go
// Filter by user
events, err := client.List(ctx, tryl.EventFilter{
	UserID: "user_123",
	Limit:  50,
})

// Filter by action with wildcard
events, err := client.List(ctx, tryl.EventFilter{
	Action: "user.*", // Matches "user.login", "user.logout", etc.
	Limit:  50,
})

// Filter by target
events, err := client.List(ctx, tryl.EventFilter{
	TargetType: "document",
	TargetID:   "doc_789",
	Limit:      50,
})
```

### Time Range Queries

```go
// Events from last 24 hours
now := time.Now()
yesterday := now.Add(-24 * time.Hour)

events, err := client.List(ctx, tryl.EventFilter{
	StartTime: &yesterday,
	EndTime:   &now,
	Order:     "desc", // Newest first
	Limit:     100,
})
```

### Metadata Queries

```go
// JSONB containment query (exact match on fields)
events, err := client.List(ctx, tryl.EventFilter{
	MetadataContains: map[string]any{
		"status":   "active",
		"priority": "high",
	},
	Limit: 50,
})

// Full-text search in metadata
events, err := client.List(ctx, tryl.EventFilter{
	MetadataSearch: "important document",
	Limit:          50,
})
```

### Pagination

**Cursor-based pagination** (recommended for large datasets):

```go
filter := tryl.EventFilter{
	Limit: 100,
	Order: "desc",
}

for {
	result, err := client.List(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}

	// Process events
	for _, event := range result.Events {
		fmt.Printf("%s: %s\n", event.ID, event.Action)
	}

	// Check if more results
	if !result.HasMore {
		break
	}

	// Use cursor for next page
	filter.Cursor = result.NextCursor
}
```

**Offset-based pagination** (simpler but less efficient):

```go
for offset := 0; ; offset += 100 {
	result, err := client.List(ctx, tryl.EventFilter{
		Offset: offset,
		Limit:  100,
	})

	if len(result.Events) == 0 {
		break
	}

	// Process events...
}
```

## Project Management

Use `NewManagementClient()` with a session token for project and API key management.

### Managing Projects

```go
// Create management client
mgmt, err := tryl.NewManagementClient(sessionToken)
if err != nil {
	log.Fatal(err)
}

// List projects
projects, err := mgmt.ListProjects(ctx)
for _, p := range projects.Projects {
	fmt.Printf("%s: %s (%s)\n", p.ID, p.Name, p.Environment)
}

// Create project
resp, err := mgmt.CreateProject(ctx, tryl.CreateProjectRequest{
	Name:        "My Project",
	Environment: "production",
})
fmt.Printf("Project ID: %s\n", resp.Project.ID)
fmt.Printf("Initial API Key: %s\n", resp.APIKey) // Only shown once!

// Delete project
err = mgmt.DeleteProject(ctx, projectID)
```

### Managing API Keys

```go
// List API keys for a project
keys, err := mgmt.ListAPIKeys(ctx, projectID)
for _, key := range keys.APIKeys {
	fmt.Printf("%s: %s (prefix: %s)\n", key.ID, key.Name, key.Prefix)
}

// Create API key with expiration
expiresAt := time.Now().Add(90 * 24 * time.Hour)
keyResp, err := mgmt.CreateAPIKey(ctx, projectID, tryl.CreateAPIKeyRequest{
	Name:        "Production Key",
	Environment: "live",
	Scopes:      []string{"events:write", "events:read"},
	ExpiresAt:   &expiresAt,
})
fmt.Printf("New API Key: %s\n", keyResp.APIKey) // Only shown once!

// Rotate API key
rotateResp, err := mgmt.RotateAPIKey(ctx, keyID, tryl.RotateAPIKeyRequest{
	NewName: "Production Key (Rotated)",
})
fmt.Printf("New Key: %s\n", rotateResp.NewAPIKey)
fmt.Printf("Old key revoked at: %s\n", rotateResp.OldKeyRevokedAt)

// Revoke API key
err = mgmt.RevokeAPIKey(ctx, keyID)
```

## Validation

The SDK validates events client-side before sending to the API:

### Validation Rules

- **UserID**: Required, max 255 characters
- **Action**: Required, max 255 characters, format: `^[a-z][a-z0-9_.]*[a-z0-9]$`
- **ActorID, TargetType, TargetID**: Optional, max 255 characters each
- **Metadata**: Must be valid JSON
- **API Key**: Format `actlog_{live|test}_*`, minimum 44 characters

### Validation Errors

```go
event := tryl.Event{
	UserID: "", // Invalid: empty
	Action: "User.Created", // Invalid: uppercase
}

_, err := client.Log(ctx, event)
if tryl.IsClientValidationError(err) {
	// This is a client-side validation error
	log.Printf("Validation failed: %v", err)
}
```

## Error Handling

The SDK provides typed errors for common scenarios:

```go
resp, err := client.Log(ctx, event)
if err != nil {
	switch {
	case tryl.IsUnauthorized(err):
		log.Println("Invalid API key")
	case tryl.IsRateLimited(err):
		log.Println("Rate limit exceeded")
	case tryl.IsValidationError(err):
		log.Println("Validation error")
	case tryl.IsClientValidationError(err):
		log.Println("Client-side validation error")
	default:
		log.Printf("Unexpected error: %v", err)
	}
}

// Check if error is retryable
if apiErr, ok := err.(*tryl.APIError); ok {
	if apiErr.IsRetryable() {
		// Retry the request
	}
}
```

### Error Types

- `tryl.APIError` - HTTP error from the API
- `tryl.ValidationError` - Client-side validation error
- `tryl.NetworkError` - Network/connection error

### Error Helpers

- `IsUnauthorized(err)` - 401 errors
- `IsRateLimited(err)` - 429 errors
- `IsValidationError(err)` - Validation errors (client or server)
- `IsClientValidationError(err)` - Client-side validation errors only
- `IsProjectNotFound(err)` - Project not found errors
- `IsKeyNotFound(err)` - API key not found errors

## Configuration

### Client Options

```go
client, err := tryl.NewClient(apiKey,
	// Custom base URL (for testing)
	tryl.WithBaseURL("https://api.example.com"),

	// Custom HTTP client
	tryl.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}),

	// Custom timeout
	tryl.WithTimeout(10 * time.Second),

	// Enable batching (interval, max batch size)
	tryl.WithBatching(1*time.Second, 50),

	// Custom retry config
	tryl.WithRetries(5, 1*time.Second, 30*time.Second),

	// Custom user agent
	tryl.WithUserAgent("MyApp/1.0"),
)
```

### Retry Configuration

The SDK automatically retries failed requests with exponential backoff:

- **Default max retries**: 3
- **Default min backoff**: 500ms
- **Default max backoff**: 10s
- **Retryable status codes**: 500-599, 429

## Examples

Complete working examples are available in the `examples/` directory:

- [`examples/basic/`](examples/basic/) - Basic event logging
- [`examples/async/`](examples/async/) - Async logging with batching
- [`examples/batching/`](examples/batching/) - Manual batch operations
- [`examples/advanced_query/`](examples/advanced_query/) - Advanced querying features
- [`examples/management/`](examples/management/) - Project and API key management

## Migration Guide

### Migrating from `WithMetadata()` to `WithMetadataValidated()`

The `WithMetadata()` method is deprecated because it silently ignores JSON marshaling errors. Use `WithMetadataValidated()` instead:

**Before (deprecated):**
```go
event := tryl.Event{
	UserID: "user_123",
	Action: "document.created",
}.WithMetadata(map[string]any{
	"title": "My Document",
})
// Marshaling errors are silently ignored ❌
```

**After (recommended):**
```go
event := tryl.Event{
	UserID: "user_123",
	Action: "document.created",
}

event, err := event.WithMetadataValidated(map[string]any{
	"title": "My Document",
})
if err != nil {
	return fmt.Errorf("invalid metadata: %w", err)
}
// Marshaling errors are returned ✅
```

For pre-validated JSON, use `SetMetadata()`:
```go
rawJSON := json.RawMessage(`{"title":"My Document"}`)
event := event.SetMetadata(rawJSON)
```

**Timeline:**
- Current version: `WithMetadata()` deprecated but still functional
- v1.0.0: `WithMetadata()` will be removed

## API Reference

Full API documentation is available at [pkg.go.dev](https://pkg.go.dev/github.com/wato/tryl_sdk).

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This SDK is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- 📧 Email: support@activitylogger.com
- 📚 Documentation: https://docs.activitylogger.com
- 🐛 Issues: https://github.com/wato/tryl_sdk/issues
