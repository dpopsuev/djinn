package agent

import (
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestMode_String(t *testing.T) {
	tests := []struct {
		m    Mode
		want string
	}{
		{ModeAsk, "ask"},
		{ModePlan, "plan"},
		{ModeAgent, "agent"},
		{ModeAuto, "auto"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Fatalf("Mode(%d).String() = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestParseMode_Valid(t *testing.T) {
	for _, name := range []string{"ask", "plan", "agent", "auto"} {
		m, err := ParseMode(name)
		if err != nil {
			t.Fatalf("ParseMode(%q): %v", name, err)
		}
		if m.String() != name {
			t.Fatalf("roundtrip: %q != %q", m.String(), name)
		}
	}
}

func TestParseMode_Invalid(t *testing.T) {
	_, err := ParseMode("yolo")
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("err = %v, want ErrInvalidMode", err)
	}
}

func TestMode_ToolsEnabled(t *testing.T) {
	if ModeAsk.ToolsEnabled() {
		t.Fatal("ask should not have tools")
	}
	if ModePlan.ToolsEnabled() {
		t.Fatal("plan should not have tools")
	}
	if !ModeAgent.ToolsEnabled() {
		t.Fatal("agent should have tools")
	}
	if !ModeAuto.ToolsEnabled() {
		t.Fatal("auto should have tools")
	}
}

func TestMode_DefaultApprove(t *testing.T) {
	call := driver.ToolCall{Name: "Read"}

	fn := ModeAuto.DefaultApprove()
	if !fn(call) {
		t.Fatal("auto mode should approve")
	}

	fn = ModeAgent.DefaultApprove()
	if fn(call) {
		t.Fatal("agent mode default should deny (interactive required)")
	}

	if ModeAsk.DefaultApprove() != nil {
		t.Fatal("ask should return nil approve")
	}
	if ModePlan.DefaultApprove() != nil {
		t.Fatal("plan should return nil approve")
	}
}

func TestMode_Next(t *testing.T) {
	if ModeAsk.Next() != ModePlan {
		t.Fatal("ask → plan")
	}
	if ModePlan.Next() != ModeAgent {
		t.Fatal("plan → agent")
	}
	if ModeAgent.Next() != ModeAuto {
		t.Fatal("agent → auto")
	}
	if ModeAuto.Next() != ModeAsk {
		t.Fatal("auto → ask")
	}
}
