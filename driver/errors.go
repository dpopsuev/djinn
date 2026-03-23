package driver

import (
	"encoding/json"
	"fmt"
)

// DriverError is a structured error from an LLM provider API.
type DriverError struct {
	StatusCode int    // HTTP status (0 if not HTTP-based)
	Retryable  bool   // true for transient errors (5xx, 429)
	Provider   string // "claude-direct", "claude-vertex", "ollama"
	RequestID  string // provider-assigned request ID, if available
	Message    string // human-readable error detail
}

func (e *DriverError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%s: %d: %s (request_id=%s)", e.Provider, e.StatusCode, e.Message, e.RequestID)
	}
	return fmt.Sprintf("%s: %d: %s", e.Provider, e.StatusCode, e.Message)
}

// IsRetryable returns whether the error represents a transient failure.
func (e *DriverError) IsRetryable() bool {
	return e.Retryable
}

// ClassifyRetryable determines if an HTTP status code is retryable.
func ClassifyRetryable(statusCode int) bool {
	switch {
	case statusCode == 429: // rate limit
		return true
	case statusCode == 408: // request timeout
		return true
	case statusCode >= 500: // server errors
		return true
	default:
		return false
	}
}

// ValidationError is returned when pre-validation catches a bad request
// before it is sent to the provider.
type ValidationError struct {
	Field   string // which field is invalid
	Message string // human-readable reason
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}

// extractRequestID parses a JSON error response body for the request_id field.
func ExtractRequestID(body []byte) string {
	var parsed struct {
		RequestID string `json:"request_id"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		return parsed.RequestID
	}
	return ""
}
