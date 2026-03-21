package signal

import (
	"testing"
	"time"
)

func TestFlagLevel_String(t *testing.T) {
	tests := []struct {
		level FlagLevel
		want  string
	}{
		{Green, "green"},
		{Yellow, "yellow"},
		{Red, "red"},
		{Black, "black"},
		{FlagLevel(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Fatalf("FlagLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestSignal_Construction(t *testing.T) {
	now := time.Now()
	s := Signal{
		Workstream: "auth",
		Level:      Yellow,
		Confidence: 0.6,
		Source:     "agent-1",
		Scope:      []string{"auth/middleware.go"},
		Category:   "security",
		Message:    "latency rising",
		Timestamp:  now,
	}
	if s.Workstream != "auth" {
		t.Fatalf("Workstream = %q, want %q", s.Workstream, "auth")
	}
	if s.Level != Yellow {
		t.Fatalf("Level = %v, want Yellow", s.Level)
	}
	if s.Confidence != 0.6 {
		t.Fatalf("Confidence = %f, want 0.6", s.Confidence)
	}
	if s.Source != "agent-1" {
		t.Fatalf("Source = %q, want %q", s.Source, "agent-1")
	}
	if len(s.Scope) != 1 || s.Scope[0] != "auth/middleware.go" {
		t.Fatalf("Scope = %v, want [auth/middleware.go]", s.Scope)
	}
	if s.Category != "security" {
		t.Fatalf("Category = %q, want %q", s.Category, "security")
	}
	if s.Message != "latency rising" {
		t.Fatalf("Message = %q, want %q", s.Message, "latency rising")
	}
	if !s.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %v, want %v", s.Timestamp, now)
	}
}

func TestSignal_ZeroValueFields(t *testing.T) {
	s := Signal{Workstream: "w1", Level: Green, Message: "ok"}
	if s.Confidence != 0 {
		t.Fatalf("zero Confidence = %f, want 0", s.Confidence)
	}
	if s.Source != "" {
		t.Fatalf("zero Source = %q, want empty", s.Source)
	}
	if s.Scope != nil {
		t.Fatalf("zero Scope = %v, want nil", s.Scope)
	}
	if s.Category != "" {
		t.Fatalf("zero Category = %q, want empty", s.Category)
	}
}
