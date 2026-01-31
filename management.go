package tryl

import (
	"time"
)

// Project represents a project in the Activity Logger system.
// Projects group API keys and events together for organizational purposes.
type Project struct {
	// ID is the unique identifier for the project (format: proj_<ulid>).
	ID string `json:"id"`
	// Name is the human-readable project name.
	Name string `json:"name"`
	// Environment indicates the project environment ("live" or "test").
	Environment string `json:"environment"`
	// CreatedAt is when the project was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the project was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateProjectRequest represents the request to create a new project.
type CreateProjectRequest struct {
	// Name is the human-readable project name (required).
	Name string `json:"name"`
	// Environment indicates the project environment: "live" or "test" (required).
	Environment string `json:"environment"`
}

// CreateProjectResponse represents the response after creating a project.
// This response includes the initial API key, which is only shown once.
type CreateProjectResponse struct {
	// Project is the created project details.
	Project Project `json:"project"`
	// APIKey is the initial API key for this project.
	// IMPORTANT: This is only returned once at creation time. Store it securely.
	APIKey string `json:"api_key"`
}

// ProjectList represents a list of projects.
type ProjectList struct {
	// Projects is the array of projects.
	Projects []Project `json:"projects"`
}

// APIKey represents metadata about an API key.
// The actual key value is never returned after initial creation.
type APIKey struct {
	// ID is the unique identifier for the API key.
	ID string `json:"id"`
	// ProjectID is the project this key belongs to.
	ProjectID string `json:"project_id"`
	// Name is a human-readable name for this key.
	Name string `json:"name"`
	// Environment indicates if this is a "live" or "test" key.
	Environment string `json:"environment"`
	// Prefix is the visible prefix of the key (e.g., "actlog_live_abc...").
	// This allows identifying keys without exposing the full value.
	Prefix string `json:"prefix"`
	// Scopes defines the permissions granted to this key.
	Scopes []string `json:"scopes"`
	// CreatedAt is when the key was created.
	CreatedAt time.Time `json:"created_at"`
	// LastUsedAt is when the key was last used (nil if never used).
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// ExpiresAt is when the key expires (nil if no expiration).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// RevokedAt is when the key was revoked (nil if not revoked).
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// CreateAPIKeyRequest represents the request to create a new API key.
type CreateAPIKeyRequest struct {
	// Name is a human-readable name for the key (required).
	Name string `json:"name"`
	// Environment indicates if this is a "live" or "test" key (required).
	Environment string `json:"environment"`
	// Scopes defines the permissions for this key (optional, defaults to all scopes).
	Scopes []string `json:"scopes,omitempty"`
	// ExpiresAt sets an expiration time for the key (optional, nil = no expiration).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents the response after creating an API key.
// The full key value is only shown once at creation time.
type CreateAPIKeyResponse struct {
	// APIKeyMetadata contains the key metadata (ID, name, etc.).
	APIKeyMetadata APIKey `json:"api_key_metadata"`
	// APIKey is the full API key value.
	// IMPORTANT: This is only returned once at creation time. Store it securely.
	APIKey string `json:"api_key"`
}

// RotateAPIKeyRequest represents the request to rotate an existing API key.
// Rotation creates a new key and revokes the old one.
type RotateAPIKeyRequest struct {
	// NewName is an optional new name for the rotated key.
	// If empty, the original name is preserved.
	NewName string `json:"new_name,omitempty"`
	// ExpiresAt sets an expiration time for the new key (optional).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// RotateAPIKeyResponse represents the response after rotating an API key.
type RotateAPIKeyResponse struct {
	// NewAPIKeyMetadata contains the new key metadata.
	NewAPIKeyMetadata APIKey `json:"new_api_key_metadata"`
	// NewAPIKey is the full new API key value.
	// IMPORTANT: This is only returned once. Store it securely.
	NewAPIKey string `json:"new_api_key"`
	// OldKeyRevokedAt is when the old key was revoked.
	OldKeyRevokedAt time.Time `json:"old_key_revoked_at"`
}

// APIKeyList represents a list of API keys for a project.
type APIKeyList struct {
	// APIKeys is the array of API key metadata.
	APIKeys []APIKey `json:"api_keys"`
}
