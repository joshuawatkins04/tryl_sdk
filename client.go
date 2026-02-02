package tryl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/joshuawatkins04/tryl_sdk/internal/transport"
	"github.com/joshuawatkins04/tryl_sdk/internal/validation"
)

// Client is the Activity Logger SDK client.
type Client struct {
	transport *transport.Transport
	retryer   *retryer
	batcher   *Batcher
	config    *clientConfig
}

// NewClient creates a new Activity Logger client with API key authentication.
// The API key is used for event logging operations.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if err := validation.ValidateAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}
	return newClientWithToken(apiKey, opts...)
}

// NewManagementClient creates a new Activity Logger client with session token authentication.
// The session token is used for project and API key management operations.
// This client can also perform event logging operations.
func NewManagementClient(sessionToken string, opts ...Option) (*Client, error) {
	if sessionToken == "" {
		return nil, fmt.Errorf("session token is required")
	}
	return newClientWithToken(sessionToken, opts...)
}

// newClientWithToken is the internal constructor shared by NewClient and NewManagementClient.
// It accepts any bearer token (API key or session token) and creates a configured client.
func newClientWithToken(token string, opts ...Option) (*Client, error) {
	config := newDefaultConfig()
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("invalid option: %w", err)
		}
	}

	httpClient := config.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.timeout,
		}
	}

	userAgent := fmt.Sprintf("activity-logger-go/%s", Version)
	if config.userAgent != "" {
		userAgent = userAgent + " " + config.userAgent
	}

	client := &Client{
		transport: &transport.Transport{
			BaseURL:    config.baseURL,
			HTTPClient: httpClient,
			APIKey:     token, // Note: APIKey field holds any bearer token
			UserAgent:  userAgent,
		},
		retryer: newRetryer(config.retryConfig),
		config:  config,
	}

	if config.batchConfig != nil {
		client.batcher = newBatcher(client, config.batchConfig)
	}

	return client, nil
}

// Log sends a single event synchronously.
// It returns the created event's ID and timestamp on success.
func (c *Client) Log(ctx context.Context, event Event) (*EventResponse, error) {
	var resp *EventResponse
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doLog(ctx, event)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doLog performs a single log request without retries.
func (c *Client) doLog(ctx context.Context, event Event) (*EventResponse, error) {
	// Validate event before sending
	if err := validation.ValidateEvent(&event); err != nil {
		// Wrap internal validation error as public ValidationError
		if fieldErr, ok := err.(*validation.FieldError); ok {
			return nil, &ValidationError{
				Field:   fieldErr.Field,
				Message: fieldErr.Message,
			}
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	req := transport.Request{
		Method: "POST",
		Path:   "/v1/events",
		Body:   event,
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var eventResp EventResponse
	if err := json.Unmarshal(resp.Body, &eventResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &eventResp, nil
}

// LogBatch sends multiple events in a single request.
func (c *Client) LogBatch(ctx context.Context, events []Event) (*batchResponse, error) {
	var resp *batchResponse
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doLogBatch(ctx, events)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doLogBatch performs a batch log request without retries.
func (c *Client) doLogBatch(ctx context.Context, events []Event) (*batchResponse, error) {
	// Validate batch size
	if len(events) == 0 {
		return nil, &ValidationError{
			Field:   "events",
			Message: "must contain at least one event",
		}
	}
	if len(events) > 100 {
		return nil, &ValidationError{
			Field:   "events",
			Message: "must contain at most 100 events",
		}
	}

	// Validate each event
	for i, event := range events {
		if err := validation.ValidateEvent(&event); err != nil {
			if fieldErr, ok := err.(*validation.FieldError); ok {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("events[%d].%s", i, fieldErr.Field),
					Message: fieldErr.Message,
				}
			}
			return nil, fmt.Errorf("event at index %d: %w", i, err)
		}
	}

	req := transport.Request{
		Method: "POST",
		Path:   "/v1/events/batch",
		Body:   batchRequest{Events: events},
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusMultiStatus {
		return nil, c.parseError(resp)
	}

	var batchResp batchResponse
	if err := json.Unmarshal(resp.Body, &batchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &batchResp, nil
}

// LogAsync queues an event for asynchronous delivery.
// It returns immediately. Use the returned channel to receive the result.
// If batching is enabled, events are accumulated and sent in bulk.
func (c *Client) LogAsync(ctx context.Context, event Event) <-chan AsyncResult {
	resultCh := make(chan AsyncResult, 1)

	if c.batcher != nil {
		c.batcher.Add(ctx, event, resultCh)
	} else {
		go func() {
			resp, err := c.Log(ctx, event)
			resultCh <- AsyncResult{Response: resp, Error: err}
			close(resultCh)
		}()
	}

	return resultCh
}

// List retrieves events matching the given filter.
func (c *Client) List(ctx context.Context, filter EventFilter) (*EventList, error) {
	var resp *EventList
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doList(ctx, filter)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doList performs a list request without retries.
func (c *Client) doList(ctx context.Context, filter EventFilter) (*EventList, error) {
	query := url.Values{}

	// Basic filters
	if filter.UserID != "" {
		query.Set("user_id", filter.UserID)
	}
	if filter.ActorID != "" {
		query.Set("actor_id", filter.ActorID)
	}
	if filter.Action != "" {
		query.Set("action", filter.Action)
	}

	// Target filters
	if filter.TargetType != "" {
		query.Set("target_type", filter.TargetType)
	}
	if filter.TargetID != "" {
		query.Set("target_id", filter.TargetID)
	}

	// Time range filters
	if filter.StartTime != nil {
		query.Set("start_time", filter.StartTime.Format(time.RFC3339))
	}
	if filter.EndTime != nil {
		query.Set("end_time", filter.EndTime.Format(time.RFC3339))
	}

	// Metadata filters
	if filter.MetadataContains != nil {
		jsonData, err := json.Marshal(filter.MetadataContains)
		if err != nil {
			return nil, &ValidationError{
				Field:   "metadata_contains",
				Message: fmt.Sprintf("failed to marshal metadata filter: %v", err),
			}
		}
		query.Set("metadata_contains", string(jsonData))
	}
	if filter.MetadataSearch != "" {
		query.Set("metadata_search", filter.MetadataSearch)
	}

	// Pagination: Cursor takes precedence over Offset
	if filter.Cursor != "" {
		query.Set("cursor", filter.Cursor)
	} else if filter.Offset > 0 {
		query.Set("offset", strconv.Itoa(filter.Offset))
	}

	// Limit
	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}

	// Order
	if filter.Order != "" {
		query.Set("order", filter.Order)
	}

	req := transport.Request{
		Method: "GET",
		Path:   "/v1/events",
		Query:  query,
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var eventList EventList
	if err := json.Unmarshal(resp.Body, &eventList); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &eventList, nil
}

// Flush sends any buffered events immediately.
// Should be called before application shutdown.
func (c *Client) Flush(ctx context.Context) error {
	if c.batcher != nil {
		return c.batcher.Flush(ctx)
	}
	return nil
}

// Close gracefully shuts down the client, flushing any pending events.
func (c *Client) Close() error {
	if c.batcher != nil {
		return c.batcher.Stop(context.Background())
	}
	return nil
}

// ========== Project Management Methods ==========

// ListProjects retrieves all projects for the authenticated user.
// Requires session token authentication (use NewManagementClient).
func (c *Client) ListProjects(ctx context.Context) (*ProjectList, error) {
	var resp *ProjectList
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doListProjects(ctx)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doListProjects performs the list projects request without retries.
func (c *Client) doListProjects(ctx context.Context) (*ProjectList, error) {
	req := transport.Request{
		Method: "GET",
		Path:   "/v1/projects",
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var projectList ProjectList
	if err := json.Unmarshal(resp.Body, &projectList); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &projectList, nil
}

// CreateProject creates a new project.
// Requires session token authentication (use NewManagementClient).
// Returns the project details and an initial API key (shown only once).
func (c *Client) CreateProject(ctx context.Context, req CreateProjectRequest) (*CreateProjectResponse, error) {
	var resp *CreateProjectResponse
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doCreateProject(ctx, req)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doCreateProject performs the create project request without retries.
func (c *Client) doCreateProject(ctx context.Context, req CreateProjectRequest) (*CreateProjectResponse, error) {
	transportReq := transport.Request{
		Method: "POST",
		Path:   "/v1/projects",
		Body:   req,
	}

	resp, err := c.transport.Do(ctx, transportReq)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var createResp CreateProjectResponse
	if err := json.Unmarshal(resp.Body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &createResp, nil
}

// DeleteProject deletes a project by ID.
// Requires session token authentication (use NewManagementClient).
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		err := c.doDeleteProject(ctx, projectID)
		if err != nil {
			lastErr = err
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return lastErr
}

// doDeleteProject performs the delete project request without retries.
func (c *Client) doDeleteProject(ctx context.Context, projectID string) error {
	req := transport.Request{
		Method: "DELETE",
		Path:   fmt.Sprintf("/v1/projects/%s", projectID),
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}

	return nil
}

// ========== API Key Management Methods ==========

// ListAPIKeys retrieves all API keys for a project.
// Requires session token authentication (use NewManagementClient).
func (c *Client) ListAPIKeys(ctx context.Context, projectID string) (*APIKeyList, error) {
	var resp *APIKeyList
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doListAPIKeys(ctx, projectID)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doListAPIKeys performs the list API keys request without retries.
func (c *Client) doListAPIKeys(ctx context.Context, projectID string) (*APIKeyList, error) {
	req := transport.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/v1/projects/%s/keys", projectID),
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var keyList APIKeyList
	if err := json.Unmarshal(resp.Body, &keyList); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &keyList, nil
}

// CreateAPIKey creates a new API key for a project.
// Requires session token authentication (use NewManagementClient).
// Returns the full API key value (shown only once).
func (c *Client) CreateAPIKey(ctx context.Context, projectID string, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	var resp *CreateAPIKeyResponse
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doCreateAPIKey(ctx, projectID, req)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doCreateAPIKey performs the create API key request without retries.
func (c *Client) doCreateAPIKey(ctx context.Context, projectID string, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	transportReq := transport.Request{
		Method: "POST",
		Path:   fmt.Sprintf("/v1/projects/%s/keys", projectID),
		Body:   req,
	}

	resp, err := c.transport.Do(ctx, transportReq)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var createResp CreateAPIKeyResponse
	if err := json.Unmarshal(resp.Body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &createResp, nil
}

// RevokeAPIKey revokes an API key by ID.
// Requires session token authentication (use NewManagementClient).
func (c *Client) RevokeAPIKey(ctx context.Context, keyID string) error {
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		err := c.doRevokeAPIKey(ctx, keyID)
		if err != nil {
			lastErr = err
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return lastErr
}

// doRevokeAPIKey performs the revoke API key request without retries.
func (c *Client) doRevokeAPIKey(ctx context.Context, keyID string) error {
	req := transport.Request{
		Method: "POST",
		Path:   fmt.Sprintf("/v1/keys/%s/revoke", keyID),
	}

	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}

	return nil
}

// RotateAPIKey rotates an API key, creating a new key and revoking the old one.
// Requires session token authentication (use NewManagementClient).
// Returns the new API key value (shown only once) and the revocation timestamp.
func (c *Client) RotateAPIKey(ctx context.Context, keyID string, req RotateAPIKeyRequest) (*RotateAPIKeyResponse, error) {
	var resp *RotateAPIKeyResponse
	var lastErr error

	err := c.retryer.do(ctx, func() error {
		r, err := c.doRotateAPIKey(ctx, keyID, req)
		if err != nil {
			lastErr = err
			return err
		}
		resp = r
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, lastErr
}

// doRotateAPIKey performs the rotate API key request without retries.
func (c *Client) doRotateAPIKey(ctx context.Context, keyID string, req RotateAPIKeyRequest) (*RotateAPIKeyResponse, error) {
	transportReq := transport.Request{
		Method: "POST",
		Path:   fmt.Sprintf("/v1/keys/%s/rotate", keyID),
		Body:   req,
	}

	resp, err := c.transport.Do(ctx, transportReq)
	if err != nil {
		return nil, &NetworkError{Op: "request", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp)
	}

	var rotateResp RotateAPIKeyResponse
	if err := json.Unmarshal(resp.Body, &rotateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &rotateResp, nil
}

// parseError converts an HTTP error response to an APIError.
func (c *Client) parseError(resp *transport.Response) error {
	errResp := transport.ParseError(resp)
	if errResp != nil {
		return &APIError{
			HTTPStatus: resp.StatusCode,
			Code:       errResp.Error.Code,
			Message:    errResp.Error.Message,
			RequestID:  resp.RequestID,
		}
	}

	return &APIError{
		HTTPStatus: resp.StatusCode,
		Code:       "unknown_error",
		Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(resp.Body)),
		RequestID:  resp.RequestID,
	}
}

// AsyncResult represents the outcome of an async log operation.
type AsyncResult struct {
	Response *EventResponse
	Error    error
}
