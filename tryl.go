// Package tryl provides a Go SDK for the Activity Logger service.
//
// The SDK allows you to log user activity events and retrieve them via a simple API.
//
// Basic usage:
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
package tryl

// Version is the SDK version.
const Version = "0.1.0"
