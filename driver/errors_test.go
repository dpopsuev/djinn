package driver

import (
	"errors"
	"testing"
)

func TestDriverError_ErrorString(t *testing.T) {
	e := &DriverError{
		StatusCode: 500,
		Retryable:  true,
		Provider:   "claude-vertex",
		Message:    "Internal server error",
	}
	got := e.Error()
	want := "claude-vertex: 500: Internal server error"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestDriverError_ErrorString_WithRequestID(t *testing.T) {
	e := &DriverError{
		StatusCode: 429,
		Retryable:  true,
		Provider:   "claude-direct",
		RequestID:  "req_abc123",
		Message:    "rate limited",
	}
	got := e.Error()
	want := "claude-direct: 429: rate limited (request_id=req_abc123)"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestDriverError_ErrorsAs(t *testing.T) {
	var wrapped error = &DriverError{
		StatusCode: 401,
		Provider:   "claude-direct",
		Message:    "unauthorized",
	}

	var de *DriverError
	if !errors.As(wrapped, &de) {
		t.Fatal("errors.As failed")
	}
	if de.StatusCode != 401 {
		t.Fatalf("StatusCode = %d, want 401", de.StatusCode)
	}
}

func TestClassifyRetryable(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{408, true},
		{422, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}
	for _, tt := range tests {
		if got := ClassifyRetryable(tt.code); got != tt.want {
			t.Errorf("ClassifyRetryable(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestDriverError_IsRetryable(t *testing.T) {
	retryable := &DriverError{StatusCode: 500, Retryable: true}
	if !retryable.IsRetryable() {
		t.Fatal("expected retryable")
	}

	permanent := &DriverError{StatusCode: 400, Retryable: false}
	if permanent.IsRetryable() {
		t.Fatal("expected not retryable")
	}
}

func TestValidationError_ErrorString(t *testing.T) {
	e := &ValidationError{Field: "model", Message: "model is required"}
	got := e.Error()
	want := "validation: model: model is required"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestExtractRequestID(t *testing.T) {
	body := []byte(`{"type":"error","error":{"type":"api_error","message":"Internal server error"},"request_id":"req_vrtx_abc123"}`)
	got := ExtractRequestID(body)
	if got != "req_vrtx_abc123" {
		t.Fatalf("ExtractRequestID = %q, want %q", got, "req_vrtx_abc123")
	}
}

func TestExtractRequestID_NoID(t *testing.T) {
	body := []byte(`{"error": "bad request"}`)
	got := ExtractRequestID(body)
	if got != "" {
		t.Fatalf("ExtractRequestID = %q, want empty", got)
	}
}

func TestExtractRequestID_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	got := ExtractRequestID(body)
	if got != "" {
		t.Fatalf("ExtractRequestID = %q, want empty", got)
	}
}
