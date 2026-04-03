package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestHookRunner_AllowExitCode0(t *testing.T) {
	hr := NewHookRunner(
		[]HookConfig{{Command: "exit 0", Tools: []string{"Edit"}}},
		nil,
		nil,
	)

	result, err := hr.Check(context.Background(), "Edit", json.RawMessage(`{"path":"foo.go"}`))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Allowed {
		t.Fatalf("Check() Allowed = false, want true")
	}
}

func TestHookRunner_DenyExitCode2(t *testing.T) {
	hr := NewHookRunner(
		[]HookConfig{{Command: `echo -n "not allowed" && exit 2`, Tools: []string{"Bash"}}},
		nil,
		nil,
	)

	result, err := hr.Check(context.Background(), "Bash", json.RawMessage(`{"command":"rm -rf /"}`))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Allowed {
		t.Fatalf("Check() Allowed = true, want false")
	}
	if result.Reason != "not allowed" {
		t.Fatalf("Check() Reason = %q, want %q", result.Reason, "not allowed")
	}
}

func TestHookRunner_WarnOtherExitCode(t *testing.T) {
	hr := NewHookRunner(
		[]HookConfig{{Command: "exit 1", Tools: []string{"Edit"}}},
		nil,
		nil,
	)

	result, err := hr.Check(context.Background(), "Edit", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Allowed {
		t.Fatalf("Check() Allowed = false, want true (non-2 exit codes should warn but allow)")
	}
}

func TestHookRunner_WildcardMatchesAll(t *testing.T) {
	hr := NewHookRunner(
		[]HookConfig{{Command: "exit 0", Tools: []string{"*"}}},
		nil,
		nil,
	)

	for _, tool := range []string{"Edit", "Read", "Bash", "Write", "Glob", "Grep"} {
		result, err := hr.Check(context.Background(), tool, json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("Check(%s) error = %v", tool, err)
		}
		if !result.Allowed {
			t.Fatalf("Check(%s) Allowed = false, want true with wildcard", tool)
		}
	}
}

func TestHookRunner_NoHooks_Passthrough(t *testing.T) {
	hr := NewHookRunner(nil, nil, nil)

	result, err := hr.Check(context.Background(), "Edit", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Allowed {
		t.Fatalf("Check() Allowed = false, want true with no hooks")
	}
}

func TestHookRunner_PostHookRuns(t *testing.T) {
	// Use a command that always succeeds. We just verify Record doesn't panic.
	hr := NewHookRunner(
		nil,
		[]HookConfig{{Command: "exit 0", Tools: []string{"Edit"}}},
		nil,
	)

	hr.Record(context.Background(), "Edit", json.RawMessage(`{}`), "output", nil, time.Second)
}

func TestHookRunner_NonMatchingTool_Skipped(t *testing.T) {
	hr := NewHookRunner(
		[]HookConfig{{Command: "exit 2", Tools: []string{"Bash"}}},
		nil,
		nil,
	)

	// Edit should not match a hook configured for Bash only
	result, err := hr.Check(context.Background(), "Edit", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Allowed {
		t.Fatalf("Check() Allowed = false, want true (hook should not match Edit)")
	}
}

func TestHookRunner_FirstDenyStopsChain(t *testing.T) {
	hr := NewHookRunner(
		[]HookConfig{
			{Command: `echo -n "first deny" && exit 2`, Tools: []string{"Edit"}},
			{Command: "exit 0", Tools: []string{"Edit"}},
		},
		nil,
		nil,
	)

	result, err := hr.Check(context.Background(), "Edit", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Allowed {
		t.Fatalf("Check() Allowed = true, want false (first deny should stop chain)")
	}
	if result.Reason != "first deny" {
		t.Fatalf("Check() Reason = %q, want %q", result.Reason, "first deny")
	}
}

func TestHookMatches(t *testing.T) {
	tests := []struct {
		name  string
		hook  HookConfig
		tool  string
		match bool
	}{
		{"exact match", HookConfig{Tools: []string{"Edit"}}, "Edit", true},
		{"wildcard", HookConfig{Tools: []string{"*"}}, "Anything", true},
		{"no match", HookConfig{Tools: []string{"Edit"}}, "Bash", false},
		{"multi tool match", HookConfig{Tools: []string{"Edit", "Write"}}, "Write", true},
		{"empty tools", HookConfig{Tools: nil}, "Edit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hookMatches(tt.hook, tt.tool); got != tt.match {
				t.Fatalf("hookMatches() = %v, want %v", got, tt.match)
			}
		})
	}
}
