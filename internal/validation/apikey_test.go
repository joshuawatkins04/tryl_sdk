package validation

import (
	"testing"
)

func TestValidateAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		apiKey  string
		wantErr error
	}{
		{
			name:    "valid live key",
			apiKey:  "actlog_live_1234567890abcdef1234567890abcdef",
			wantErr: nil,
		},
		{
			name:    "valid test key",
			apiKey:  "actlog_test_1234567890abcdef1234567890abcdef",
			wantErr: nil,
		},
		{
			name:    "valid live key exactly 44 chars",
			apiKey:  "actlog_live_12345678901234567890123456789012",
			wantErr: nil,
		},
		{
			name:    "valid test key exactly 44 chars",
			apiKey:  "actlog_test_1234567890123456789012345678901",
			wantErr: nil,
		},
		{
			name:    "valid live key with longer random part",
			apiKey:  "actlog_live_1234567890abcdef1234567890abcdef1234567890",
			wantErr: nil,
		},
		{
			name:    "empty key",
			apiKey:  "",
			wantErr: ErrAPIKeyEmpty,
		},
		{
			name:    "invalid prefix",
			apiKey:  "invalid_1234567890abcdef1234567890abcdef1234567890",
			wantErr: ErrAPIKeyInvalidFormat,
		},
		{
			name:    "no prefix",
			apiKey:  "1234567890abcdef1234567890abcdef1234567890",
			wantErr: ErrAPIKeyInvalidFormat,
		},
		{
			name:    "live key too short",
			apiKey:  "actlog_live_123",
			wantErr: ErrAPIKeyTooShort,
		},
		{
			name:    "test key too short",
			apiKey:  "actlog_test_123",
			wantErr: ErrAPIKeyTooShort,
		},
		{
			name:    "live key 43 chars (one short)",
			apiKey:  "actlog_live_1234567890123456789012345678901",
			wantErr: ErrAPIKeyTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAPIKey(tt.apiKey)
			if err != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsLiveKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{
			name:   "live key",
			apiKey: "actlog_live_1234567890abcdef1234567890abcdef",
			want:   true,
		},
		{
			name:   "test key",
			apiKey: "actlog_test_1234567890abcdef1234567890abcdef",
			want:   false,
		},
		{
			name:   "invalid key",
			apiKey: "invalid_key",
			want:   false,
		},
		{
			name:   "empty key",
			apiKey: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsLiveKey(tt.apiKey)
			if got != tt.want {
				t.Errorf("IsLiveKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTestKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{
			name:   "test key",
			apiKey: "actlog_test_1234567890abcdef1234567890abcdef",
			want:   true,
		},
		{
			name:   "live key",
			apiKey: "actlog_live_1234567890abcdef1234567890abcdef",
			want:   false,
		},
		{
			name:   "invalid key",
			apiKey: "invalid_key",
			want:   false,
		},
		{
			name:   "empty key",
			apiKey: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsTestKey(tt.apiKey)
			if got != tt.want {
				t.Errorf("IsTestKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
