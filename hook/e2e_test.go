package hook

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/troupe/signal"
)

// TestE2E_UnifiedHookFiresOnAllEventTypes proves the full pipeline:
// YAML hooks → EventDispatcher → pre_tool_use denies → post_tool_use emits → event hook reacts.
func TestE2E_UnifiedHookFiresOnAllEventTypes(t *testing.T) {
	eventLog := signal.NewMemLog()

	hooks := []Hook{
		{
			Name:   "deny-bash",
			On:     PhasePreToolUse,
			Match:  Matcher{Tool: "Bash"},
			Action: Action{Deny: "Bash not allowed"},
		},
		{
			Name:   "log-writes",
			On:     PhasePostToolUse,
			Match:  Matcher{Tool: "Write"},
			Action: Action{Emit: "file.written"},
		},
		{
			Name:   "react-to-violation",
			On:     PhaseEvent,
			Match:  Matcher{Kind: "agent.violation"},
			Action: Action{Emit: "violation.acknowledged"},
		},
	}

	dispatcher := New(hooks, eventLog)

	// 1. Pre-tool: Bash should be denied.
	verdict, err := dispatcher.Check(context.Background(), "Bash", nil)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if verdict.Allowed {
		t.Fatal("Bash should be denied by deny-bash hook")
	}
	if verdict.Reason != "Bash not allowed" {
		t.Errorf("reason = %q", verdict.Reason)
	}

	// 2. Pre-tool: Read should be allowed (no matching hook).
	verdict, err = dispatcher.Check(context.Background(), "Read", nil)
	if err != nil {
		t.Fatalf("Check Read: %v", err)
	}
	if !verdict.Allowed {
		t.Fatal("Read should be allowed")
	}

	// 3. Post-tool: Write should emit file.written event.
	dispatcher.Record(context.Background(), "Write", nil, "ok", nil, 10*time.Millisecond)

	events := eventLog.Since(-1)
	var found bool
	for _, e := range events {
		if e.Kind == "file.written" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("post_tool_use hook should have emitted file.written event")
	}

	// 4. Async event: emit agent.violation → hook reacts with violation.acknowledged.
	eventLog.Emit(signal.Event{Kind: "agent.violation", Source: "gensec"})

	// Give async handler time to fire.
	time.Sleep(10 * time.Millisecond)

	events = eventLog.Since(-1)
	var acked bool
	for _, e := range events {
		if e.Kind == "violation.acknowledged" {
			acked = true
			break
		}
	}
	if !acked {
		t.Fatal("event hook should have emitted violation.acknowledged")
	}
}

// TestE2E_ParseHooksFromYAML proves YAML → hooks → dispatcher roundtrip.
func TestE2E_ParseHooksFromYAML(t *testing.T) {
	yaml := `
hooks:
  - name: deny-dangerous
    on: pre_tool_use
    match:
      tool: Bash
    action:
      deny: "not allowed"
  - name: log-tool
    on: post_tool_use
    action:
      emit: tool.logged
  - name: spawn-on-error
    on: event
    match:
      kind: agent.error
    action:
      spawn_slot: reviewer
`
	cfg, err := ParseHooks([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseHooks: %v", err)
	}
	if len(cfg.Hooks) != 3 {
		t.Fatalf("hooks = %d, want 3", len(cfg.Hooks))
	}

	// Wire into dispatcher.
	eventLog := signal.NewMemLog()
	spawned := make(chan string, 1)
	dispatcher := New(cfg.Hooks, eventLog,
		WithSpawnFunc(func(_ context.Context, role string) error {
			spawned <- role
			return nil
		}),
	)

	// Verify deny works.
	v, _ := dispatcher.Check(context.Background(), "Bash", nil)
	if v.Allowed {
		t.Fatal("Bash should be denied")
	}

	// Verify event hook spawns.
	eventLog.Emit(signal.Event{Kind: "agent.error", Source: "coder"})

	select {
	case role := <-spawned:
		if role != "reviewer" {
			t.Errorf("spawned %q, want reviewer", role)
		}
	case <-time.After(time.Second):
		t.Fatal("spawn_slot action did not fire")
	}
}

// TestE2E_ScopedHook proves scope filtering.
func TestE2E_ScopedHook(t *testing.T) {
	eventLog := signal.NewMemLog()

	hooks := []Hook{
		{
			Name:   "coder-only",
			On:     PhaseEvent,
			Match:  Matcher{Kind: "tool.call"},
			Action: Action{Emit: "coder.tool.tracked"},
			Scope:  "coder",
		},
	}

	_ = New(hooks, eventLog)

	// Event from coder → hook fires.
	eventLog.Emit(signal.Event{Kind: "tool.call", Source: "coder-1"})
	time.Sleep(10 * time.Millisecond)

	// Event from gensec → hook does NOT fire.
	eventLog.Emit(signal.Event{Kind: "tool.call", Source: "gensec"})
	time.Sleep(10 * time.Millisecond)

	events := eventLog.Since(-1)
	tracked := 0
	for _, e := range events {
		if e.Kind == "coder.tool.tracked" {
			tracked++
		}
	}
	if tracked != 1 {
		t.Fatalf("expected 1 coder.tool.tracked, got %d", tracked)
	}
}
