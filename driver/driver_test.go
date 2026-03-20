package driver

import "testing"

func TestDriverConfig_Construction(t *testing.T) {
	cfg := DriverConfig{
		Model:       "claude-opus-4-6",
		MaxTokens:   4096,
		Temperature: 0.7,
	}
	if cfg.Model != "claude-opus-4-6" {
		t.Fatalf("Model = %q, want %q", cfg.Model, "claude-opus-4-6")
	}
	if cfg.MaxTokens != 4096 {
		t.Fatalf("MaxTokens = %d, want 4096", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.7 {
		t.Fatalf("Temperature = %f, want 0.7", cfg.Temperature)
	}
}

func TestMessage_Construction(t *testing.T) {
	msg := Message{Role: "assistant", Content: "hello"}
	if msg.Role != "assistant" {
		t.Fatalf("Role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Content != "hello" {
		t.Fatalf("Content = %q, want %q", msg.Content, "hello")
	}
}
