package validation

import (
	"errors"
	"strings"
)

var (
	// ErrAPIKeyEmpty indicates the API key is missing.
	ErrAPIKeyEmpty = errors.New("API key is required")
	// ErrAPIKeyInvalidFormat indicates the API key format is invalid.
	ErrAPIKeyInvalidFormat = errors.New("API key must start with actlog_live_ or actlog_test_")
	// ErrAPIKeyTooShort indicates the API key is too short.
	ErrAPIKeyTooShort = errors.New("API key must be at least 44 characters")
)

// ValidateAPIKey validates the API key format according to server-side rules.
// The API key must:
// - Start with "actlog_live_" or "actlog_test_"
// - Be at least 44 characters long (prefix 12/13 chars + 32 random chars)
//
// Server validation source: internal/middleware/auth.go:186-194
//
// Returns nil if valid, or a descriptive error if invalid.
func ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return ErrAPIKeyEmpty
	}

	if !strings.HasPrefix(apiKey, "actlog_live_") && !strings.HasPrefix(apiKey, "actlog_test_") {
		return ErrAPIKeyInvalidFormat
	}

	if len(apiKey) < 44 {
		return ErrAPIKeyTooShort
	}

	return nil
}

// IsLiveKey returns true if the API key is a live (production) key.
func IsLiveKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, "actlog_live_")
}

// IsTestKey returns true if the API key is a test key.
func IsTestKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, "actlog_test_")
}
