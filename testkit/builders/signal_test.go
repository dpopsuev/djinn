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
		WithMessage("tests failing").
		WithTimestamp(now).
		Build()

	if s.Workstream != "auth" {
		t.Fatalf("Workstream = %q, want %q", s.Workstream, "auth")
	}
	if s.Level != signal.Red {
		t.Fatalf("Level = %v, want Red", s.Level)
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
}
