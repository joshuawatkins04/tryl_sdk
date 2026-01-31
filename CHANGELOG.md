# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Type Safety & Validation
- **Client-side event validation** matching server-side rules to fail fast
  - UserID: required, max 255 characters
  - Action: required, max 255 characters, format `^[a-z][a-z0-9_.]*[a-z0-9]$`
  - Optional fields: max 255 characters each
  - Metadata: valid JSON validation
- **API key format validation** at client construction (`actlog_{live|test}_*`, min 44 chars)
- **New metadata methods** on `Event`:
  - `WithMetadataValidated(map[string]any) (Event, error)` - Returns errors for invalid JSON
  - `SetMetadata(json.RawMessage) Event` - For pre-validated JSON
  - Getter methods for validation interface compatibility
- **New error types**:
  - `ValidationError` - Client-side validation errors with field details
  - `ErrInvalidAPIKey` - Sentinel error for invalid API key format
  - `IsClientValidationError(err)` - Helper to distinguish client/server validation errors
- **Internal validation package** (`internal/validation/`) with comprehensive test coverage

#### Enhanced Event Queries
- **Extended `EventFilter` with 8 new fields**:
  - `TargetType` and `TargetID` - Filter by target resource
  - `StartTime` and `EndTime` - Time range queries with RFC3339 formatting
  - `MetadataContains` - JSONB containment queries
  - `MetadataSearch` - Full-text search in metadata
  - `Cursor` - Efficient cursor-based pagination (takes precedence over Offset)
  - `Order` - Sort order ("asc" or "desc")
- **Updated `EventList` response**:
  - `NextCursor` field for cursor-based pagination
  - `Total` only populated with offset-based pagination
- **Wildcard action filters**: Support for `org.*` and `*.created` patterns

#### Project & API Key Management
- **New management client constructor**:
  - `NewManagementClient(sessionToken, ...Option) (*Client, error)`
  - Uses session token authentication for management operations
  - Same `Client` struct works with both API key and session token
- **7 new management methods**:
  - `ListProjects(ctx) (*ProjectList, error)`
  - `CreateProject(ctx, CreateProjectRequest) (*CreateProjectResponse, error)`
  - `DeleteProject(ctx, projectID string) error`
  - `ListAPIKeys(ctx, projectID string) (*APIKeyList, error)`
  - `CreateAPIKey(ctx, projectID string, CreateAPIKeyRequest) (*CreateAPIKeyResponse, error)`
  - `RevokeAPIKey(ctx, keyID string) error`
  - `RotateAPIKey(ctx, keyID string, RotateAPIKeyRequest) (*RotateAPIKeyResponse, error)`
- **New management types** in `management.go`:
  - Project types: `Project`, `CreateProjectRequest`, `CreateProjectResponse`, `ProjectList`
  - API Key types: `APIKey`, `CreateAPIKeyRequest`, `CreateAPIKeyResponse`, `RotateAPIKeyRequest`, `RotateAPIKeyResponse`, `APIKeyList`
- **New error codes and helpers**:
  - `ErrCodeProjectNotFound`, `ErrCodeKeyNotFound`
  - `ErrProjectNotFound`, `ErrKeyNotFound` - Sentinel errors
  - `IsProjectNotFound(err)`, `IsKeyNotFound(err)` - Helper functions

#### Documentation & Examples
- **Comprehensive README.md** with quick start, complete examples, and migration guide
- **Example programs**:
  - `examples/advanced_query/main.go` - 8 query examples (wildcards, time ranges, metadata, pagination)
  - `examples/management/main.go` - Complete project/key management workflow
- **Migration guide** for deprecated `WithMetadata()` method

### Changed

- **Refactored client construction**: `NewClient()` now shares logic with `NewManagementClient()` via internal `newClientWithToken()`
- **Enhanced validation**: All events validated before network calls to catch errors early
- **Improved error messages**: Validation errors include field names and clear descriptions

### Deprecated

- **`Event.WithMetadata(map[string]any) Event`** - Silently ignores JSON marshaling errors
  - Use `WithMetadataValidated(map[string]any) (Event, error)` instead
  - Will be removed in v1.0.0
- **`EventFilter.Offset`** - Less efficient than cursor-based pagination
  - Use `Cursor` for better performance with large datasets
  - Still functional for backward compatibility

### Fixed

- **Metadata marshaling errors** now properly returned to callers via `WithMetadataValidated()`
- **Invalid API keys** rejected at client construction instead of first API call

### Security

- **Client-side validation** prevents sending invalid data to the API
- **API key format validation** catches malformed keys early
- **Full API keys only returned once** at creation/rotation

## [0.1.0] - 2026-01-30

### Added

- Initial release of Tryl Go SDK
- `Client` with `Log`, `LogAsync`, `LogBatch`, and `List` methods
- Event batching support with configurable batch size and flush interval
- Automatic retry with exponential backoff for transient errors
- Typed errors (`APIError`, `NetworkError`) with helper functions
- Configuration options: `WithBaseURL`, `WithTimeout`, `WithRetry`, `WithBatching`, `WithHTTPClient`, `WithUserAgent`
- Examples for basic usage, async logging, and batching
