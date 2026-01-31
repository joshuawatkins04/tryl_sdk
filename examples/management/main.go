package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wato/tryl_sdk"
)

func main() {
	// Get session token from environment
	// This is obtained from your authentication provider (e.g., Stytch session)
	sessionToken := os.Getenv("ACTLOG_SESSION_TOKEN")
	if sessionToken == "" {
		log.Fatal("ACTLOG_SESSION_TOKEN environment variable is required")
	}

	// Create management client with session token
	client, err := tryl.NewManagementClient(sessionToken)
	if err != nil {
		log.Fatalf("Failed to create management client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: List all projects
	fmt.Println("=== Example 1: List Projects ===")
	projects, err := client.ListProjects(ctx)
	if err != nil {
		log.Fatalf("Failed to list projects: %v", err)
	}
	fmt.Printf("Found %d projects:\n", len(projects.Projects))
	for _, project := range projects.Projects {
		fmt.Printf("  - %s: %s (%s)\n", project.ID, project.Name, project.Environment)
	}
	fmt.Println()

	// Example 2: Create a new project
	fmt.Println("=== Example 2: Create Project ===")
	createResp, err := client.CreateProject(ctx, tryl.CreateProjectRequest{
		Name:        "My New Project",
		Environment: "test",
	})
	if err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}
	fmt.Printf("Created project: %s\n", createResp.Project.ID)
	fmt.Printf("Initial API Key: %s\n", createResp.APIKey)
	fmt.Println("⚠️  IMPORTANT: Store this API key securely - it's only shown once!")
	fmt.Println()

	projectID := createResp.Project.ID

	// Example 3: List API keys for a project
	fmt.Println("=== Example 3: List API Keys ===")
	keys, err := client.ListAPIKeys(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to list API keys: %v", err)
	}
	fmt.Printf("Found %d API keys for project %s:\n", len(keys.APIKeys), projectID)
	for _, key := range keys.APIKeys {
		fmt.Printf("  - %s: %s (%s) - Prefix: %s\n",
			key.ID, key.Name, key.Environment, key.Prefix)
		if key.LastUsedAt != nil {
			fmt.Printf("    Last used: %s\n", key.LastUsedAt.Format(time.RFC3339))
		}
		if key.ExpiresAt != nil {
			fmt.Printf("    Expires: %s\n", key.ExpiresAt.Format(time.RFC3339))
		}
	}
	fmt.Println()

	// Example 4: Create a new API key with expiration
	fmt.Println("=== Example 4: Create API Key with Expiration ===")
	expiresAt := time.Now().Add(90 * 24 * time.Hour) // 90 days
	keyResp, err := client.CreateAPIKey(ctx, projectID, tryl.CreateAPIKeyRequest{
		Name:        "Production Key",
		Environment: "live",
		Scopes:      []string{"events:write", "events:read"},
		ExpiresAt:   &expiresAt,
	})
	if err != nil {
		log.Fatalf("Failed to create API key: %v", err)
	}
	fmt.Printf("Created API key: %s\n", keyResp.APIKeyMetadata.ID)
	fmt.Printf("Full API Key: %s\n", keyResp.APIKey)
	fmt.Printf("Expires: %s\n", keyResp.APIKeyMetadata.ExpiresAt.Format(time.RFC3339))
	fmt.Println("⚠️  IMPORTANT: Store this API key securely - it's only shown once!")
	fmt.Println()

	keyID := keyResp.APIKeyMetadata.ID

	// Example 5: Rotate an API key
	fmt.Println("=== Example 5: Rotate API Key ===")
	rotateResp, err := client.RotateAPIKey(ctx, keyID, tryl.RotateAPIKeyRequest{
		NewName: "Production Key (Rotated)",
	})
	if err != nil {
		log.Fatalf("Failed to rotate API key: %v", err)
	}
	fmt.Printf("Rotated API key: %s → %s\n",
		keyID, rotateResp.NewAPIKeyMetadata.ID)
	fmt.Printf("New API Key: %s\n", rotateResp.NewAPIKey)
	fmt.Printf("Old key revoked at: %s\n", rotateResp.OldKeyRevokedAt.Format(time.RFC3339))
	fmt.Println("⚠️  IMPORTANT: Update your application to use the new API key!")
	fmt.Println()

	newKeyID := rotateResp.NewAPIKeyMetadata.ID

	// Example 6: Revoke an API key
	fmt.Println("=== Example 6: Revoke API Key ===")
	err = client.RevokeAPIKey(ctx, newKeyID)
	if err != nil {
		log.Fatalf("Failed to revoke API key: %v", err)
	}
	fmt.Printf("Revoked API key: %s\n", newKeyID)
	fmt.Println()

	// Example 7: Delete a project
	fmt.Println("=== Example 7: Delete Project ===")
	err = client.DeleteProject(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to delete project: %v", err)
	}
	fmt.Printf("Deleted project: %s\n", projectID)
	fmt.Println()

	// Example 8: Using management client for event logging
	fmt.Println("=== Example 8: Event Logging with Management Client ===")
	fmt.Println("Note: The management client can also log events!")

	// First, create a project with an API key for logging
	logProject, err := client.CreateProject(ctx, tryl.CreateProjectRequest{
		Name:        "Logging Project",
		Environment: "test",
	})
	if err != nil {
		log.Fatalf("Failed to create logging project: %v", err)
	}

	// Create a regular client with the API key for production event logging
	logClient, err := tryl.NewClient(logProject.APIKey)
	if err != nil {
		log.Fatalf("Failed to create log client: %v", err)
	}
	defer logClient.Close()

	// Log an event
	event := tryl.Event{
		UserID: "user_123",
		Action: "project.created",
	}
	eventResp, err := logClient.Log(ctx, event)
	if err != nil {
		log.Fatalf("Failed to log event: %v", err)
	}
	fmt.Printf("Logged event: %s at %s\n", eventResp.ID, eventResp.Timestamp.Format(time.RFC3339))
	fmt.Println()

	// Clean up
	err = client.DeleteProject(ctx, logProject.Project.ID)
	if err != nil {
		log.Fatalf("Failed to delete logging project: %v", err)
	}

	fmt.Println("All management examples completed successfully!")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("1. Use NewManagementClient() with session token for project/key management")
	fmt.Println("2. Use NewClient() with API key for event logging in production")
	fmt.Println("3. API keys are only shown once - store them securely!")
	fmt.Println("4. Rotate keys regularly for security")
	fmt.Println("5. Set expiration dates on API keys when possible")
}
