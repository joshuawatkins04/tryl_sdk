package tryl

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://tryl.fly.dev"
	defaultTimeout = 10 * time.Second
)

// HTTPDoer is an interface for HTTP operations (for testing).
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Option configures the Client.
type Option func(*clientConfig) error

// clientConfig holds internal configuration.
type clientConfig struct {
	baseURL     string
	httpClient  HTTPDoer
	retryConfig *RetryConfig
	batchConfig *BatchConfig
	userAgent   string
	timeout     time.Duration
}

// newDefaultConfig returns the default client configuration.
func newDefaultConfig() *clientConfig {
	return &clientConfig{
		baseURL:     defaultBaseURL,
		timeout:     defaultTimeout,
		retryConfig: defaultRetryConfig(),
	}
}

// WithBaseURL sets a custom API base URL.
// Default: "https://tryl.fly.dev"
func WithBaseURL(url string) Option {
	return func(c *clientConfig) error {
		if url == "" {
			return errors.New("base URL cannot be empty")
		}
		c.baseURL = strings.TrimSuffix(url, "/")
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client.
// Default: http.DefaultClient with configured timeout
func WithHTTPClient(client HTTPDoer) Option {
	return func(c *clientConfig) error {
		if client == nil {
			return errors.New("HTTP client cannot be nil")
		}
		c.httpClient = client
		return nil
	}
}

// WithTimeout sets the request timeout.
// Default: 10 seconds
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) error {
		if d <= 0 {
			return errors.New("timeout must be positive")
		}
		c.timeout = d
		return nil
	}
}

// WithRetry configures retry behavior.
// Default: 3 retries with exponential backoff (base 1s, max 30s)
func WithRetry(config RetryConfig) Option {
	return func(c *clientConfig) error {
		if config.MaxAttempts < 0 {
			return errors.New("max attempts cannot be negative")
		}
		c.retryConfig = &config
		return nil
	}
}

// WithoutRetry disables automatic retries.
func WithoutRetry() Option {
	return func(c *clientConfig) error {
		c.retryConfig = &RetryConfig{MaxAttempts: 1}
		return nil
	}
}

// WithBatching enables event batching.
// Events are accumulated and sent in bulk for improved throughput.
func WithBatching(config BatchConfig) Option {
	return func(c *clientConfig) error {
		if config.MaxBatchSize <= 0 {
			return errors.New("max batch size must be positive")
		}
		c.batchConfig = &config
		return nil
	}
}

// WithUserAgent sets a custom User-Agent suffix.
// The SDK will prepend its own identifier.
func WithUserAgent(ua string) Option {
	return func(c *clientConfig) error {
		c.userAgent = ua
		return nil
	}
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the initial request).
	// Set to 1 to disable retries. Default: 3
	MaxAttempts int

	// BaseDelay is the initial delay before the first retry.
	// Default: 1 second
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Default: 30 seconds
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases.
	// Default: 2.0
	Multiplier float64

	// JitterFactor adds randomness to delays (0.0 to 1.0).
	// Default: 0.2 (20% jitter)
	JitterFactor float64
}

// defaultRetryConfig returns the default retry configuration.
func defaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		BaseDelay:    1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
}

// BatchConfig configures event batching behavior.
type BatchConfig struct {
	// MaxBatchSize is the maximum number of events per batch.
	// Default: 100
	MaxBatchSize int

	// FlushInterval is how often to flush pending events.
	// Default: 5 seconds
	FlushInterval time.Duration

	// MaxPendingEvents is the maximum events that can be queued.
	// If exceeded, LogAsync will block until space is available.
	// Default: 10000
	MaxPendingEvents int

	// OnError is called when a batch fails (optional).
	OnError func(events []Event, err error)
}

// defaultBatchConfig returns the default batch configuration.
func defaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxBatchSize:     100,
		FlushInterval:    5 * time.Second,
		MaxPendingEvents: 10000,
	}
}
