// Package tryl provides a Go SDK for the Activity Logger service.
//
// The SDK allows you to log user activity events, query them with advanced filters,
// and manage projects and API keys programmatically.
//
// # Event Logging
//
// Basic usage with API key:
//
//	client, err := tryl.NewClient("actlog_live_xxxxx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	resp, err := client.Log(ctx, tryl.Event{
//	    UserID: "user_123",
//	    Action: "document.created",
//	})
//
// # Project Management
//
// For project and API key management, use a session token:
//
//	mgmt, err := tryl.NewManagementClient(sessionToken)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	projects, err := mgmt.ListProjects(ctx)
//
// # Querying Events
//
// Advanced filtering with time ranges, metadata, and pagination:
//
//	events, err := client.List(ctx, tryl.EventFilter{
//	    Action:     "user.*", // Wildcard support
//	    StartTime:  &yesterday,
//	    EndTime:    &now,
//	    Cursor:     nextCursor, // Efficient pagination
//	    Limit:      100,
//	})
package tryl

// Version is the SDK version.
const Version = "0.1.0"
