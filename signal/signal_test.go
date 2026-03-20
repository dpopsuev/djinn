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
		Message:    "latency rising",
		Timestamp:  now,
	}
	if s.Workstream != "auth" {
		t.Fatalf("Workstream = %q, want %q", s.Workstream, "auth")
	}
	if s.Level != Yellow {
		t.Fatalf("Level = %v, want Yellow", s.Level)
	}
	if s.Message != "latency rising" {
		t.Fatalf("Message = %q, want %q", s.Message, "latency rising")
	}
	if !s.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %v, want %v", s.Timestamp, now)
	}
}
