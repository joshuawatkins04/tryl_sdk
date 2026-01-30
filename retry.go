package tryl

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// retryer handles retry logic with exponential backoff.
type retryer struct {
	config *RetryConfig
}

// newRetryer creates a retryer with the given configuration.
func newRetryer(config *RetryConfig) *retryer {
	if config == nil {
		config = defaultRetryConfig()
	}
	if config.BaseDelay == 0 {
		config.BaseDelay = 1 * time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Multiplier == 0 {
		config.Multiplier = 2.0
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	return &retryer{config: config}
}

// do executes the operation with retries.
func (r *retryer) do(ctx context.Context, op func() error) error {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled: %w", err)
		}

		lastErr = op()
		if lastErr == nil {
			return nil
		}

		if !r.isRetryable(lastErr) {
			return lastErr
		}

		if attempt < r.config.MaxAttempts-1 {
			delay := r.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while waiting for retry: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateDelay computes the delay for a given attempt with jitter.
func (r *retryer) calculateDelay(attempt int) time.Duration {
	delay := float64(r.config.BaseDelay) * math.Pow(r.config.Multiplier, float64(attempt))

	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	if r.config.JitterFactor > 0 {
		jitter := delay * r.config.JitterFactor * (rand.Float64()*2 - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

// isRetryable determines if an error should be retried.
func (r *retryer) isRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.IsTemporary()
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return false
}
