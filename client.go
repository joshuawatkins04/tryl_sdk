package tryl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/wato/tryl_sdk/internal/transport"
)

// Client is the Activity Logger SDK client.
type Client struct {
	transport *transport.Transport
	retryer   *retryer
	batcher   *Batcher
	config    *clientConfig
}

// NewClient creates a new Activity Logger client.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

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
			APIKey:     apiKey,
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
	if filter.UserID != "" {
		query.Set("user_id", filter.UserID)
	}
	if filter.ActorID != "" {
		query.Set("actor_id", filter.ActorID)
	}
	if filter.Action != "" {
		query.Set("action", filter.Action)
	}
	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		query.Set("offset", strconv.Itoa(filter.Offset))
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
