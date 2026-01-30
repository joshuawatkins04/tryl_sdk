// Package transport provides HTTP transport utilities for the SDK.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Request represents an HTTP request to be made.
type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Body    any
	Headers map[string]string
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	RequestID  string
}

// Transport handles HTTP communication with the API.
type Transport struct {
	BaseURL    string
	HTTPClient HTTPDoer
	APIKey     string
	UserAgent  string
}

// HTTPDoer is an interface for HTTP operations.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Do executes an HTTP request and returns the response.
func (t *Transport) Do(ctx context.Context, req Request) (*Response, error) {
	fullURL := t.BaseURL + req.Path
	if len(req.Query) > 0 {
		fullURL += "?" + req.Query.Encode()
	}

	var bodyReader io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+t.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", t.UserAgent)

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := t.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
		RequestID:  resp.Header.Get("X-Request-ID"),
	}, nil
}

// ErrorResponse is the API error response format.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ParseError parses an error response from the API.
func ParseError(resp *Response) *ErrorResponse {
	var errResp ErrorResponse
	if err := json.Unmarshal(resp.Body, &errResp); err != nil {
		return nil
	}
	return &errResp
}
