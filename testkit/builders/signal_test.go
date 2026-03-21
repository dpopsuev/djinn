package builders

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

func TestSignalBuilder_Fluent(t *testing.T) {
	now := time.Now()
	s := NewSignal("auth").
		WithLevel(signal.Red).
		WithConfidence(0.3).
		WithSource("agent-1").
		WithScope("auth/middleware.go", "auth/handler.go").
		WithCategory("security").
		WithMessage("tests failing").
		WithTimestamp(now).
		Build()

	if s.Workstream != "auth" {
		t.Fatalf("Workstream = %q, want %q", s.Workstream, "auth")
	}
	if s.Level != signal.Red {
		t.Fatalf("Level = %v, want Red", s.Level)
	}
	if s.Confidence != 0.3 {
		t.Fatalf("Confidence = %f, want 0.3", s.Confidence)
	}
	if s.Source != "agent-1" {
		t.Fatalf("Source = %q, want %q", s.Source, "agent-1")
	}
	if len(s.Scope) != 2 || s.Scope[0] != "auth/middleware.go" {
		t.Fatalf("Scope = %v, want [auth/middleware.go auth/handler.go]", s.Scope)
	}
	if s.Category != "security" {
		t.Fatalf("Category = %q, want %q", s.Category, "security")
	}
	if s.Message != "tests failing" {
		t.Fatalf("Message = %q, want %q", s.Message, "tests failing")
	}
	if !s.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %v, want %v", s.Timestamp, now)
	}
}

func TestSignalBuilder_Defaults(t *testing.T) {
	s := NewSignal("w1").Build()
	if s.Level != signal.Green {
		t.Fatalf("default Level = %v, want Green", s.Level)
	}
	if s.Timestamp.IsZero() {
		t.Fatal("default Timestamp is zero")
	}
	if s.Confidence != 0 {
		t.Fatalf("default Confidence = %f, want 0", s.Confidence)
	}
}
