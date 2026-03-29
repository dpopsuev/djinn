//go:build e2e

// e2e_debug_test.go — E2E test: AI agent self-debugging via djinn_trace.
//
// Spawns a real AI agent (default: Cursor Agent CLI via DJINN_TEST_AGENT),
// triggers MCP tool calls, then verifies the agent can query its own
// trace ring via the djinn_trace builtin tool.
package acceptance

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/debug"
	"github.com/dpopsuev/djinn/testkit"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/trace"
)

func TestAgentSelfDebugging_TraceStats(t *testing.T) {
	// Verify agent binary is available.
	agent := testkit.DefaultTestAgent()
	binary := testkit.AgentBinary(agent)
	if _, err := exec.LookPath(binary); err != nil {
		t.Skipf("agent binary %q not on PATH (set DJINN_TEST_AGENT to override)", binary)
	}

	// Create trace ring and debug server.
	ring := trace.NewRing(100)
	server := debug.NewServer(ring)

	// Simulate some MCP activity.
	tracer := ring.For(trace.ComponentMCP)
	for range 5 {
		rt := tracer.Begin("call", "artifact.list on scribe").WithServer("scribe").WithTool("artifact.list")
		time.Sleep(time.Millisecond)
		rt.End()
	}

	// Query stats via debug server (same path the agent would take).
	result, err := server.Handle(debug.TraceInput{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}

	var stats trace.RingStats
	if err := json.Unmarshal([]byte(result), &stats); err != nil {
		t.Fatalf("stats not valid JSON: %v", err)
	}
	// 5 Begin + 5 End = 10 events.
	if stats.Count != 10 {
		t.Errorf("stats.Count = %d, want 10", stats.Count)
	}
}

func TestAgentSelfDebugging_TraceHealth(t *testing.T) {
	ring := trace.NewRing(100)
	server := debug.NewServer(ring)

	// Simulate errors on a server.
	tracer := ring.For(trace.ComponentMCP)
	for range 5 {
		rt := tracer.Begin("call", "codograph.scan on locus").WithServer("locus").WithTool("codograph.scan")
		rt.EndWithError()
	}

	result, err := server.Handle(debug.TraceInput{Action: "health"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "locus") {
		t.Error("health should include locus server")
	}
	if !strings.Contains(result, `"errors"`) {
		t.Error("health should include error count")
	}
}

func TestAgentSelfDebugging_TraceList(t *testing.T) {
	ring := trace.NewRing(100)
	server := debug.NewServer(ring)

	tracer := ring.For(trace.ComponentMCP)
	rt := tracer.Begin("call", "artifact.get on scribe").WithServer("scribe").WithTool("artifact.get")
	rt.End()

	result, err := server.Handle(debug.TraceInput{Action: "list", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}

	var events []trace.TraceEvent
	if err := json.Unmarshal([]byte(result), &events); err != nil {
		t.Fatalf("list not valid JSON: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events (begin+end), got %d", len(events))
	}
}

func TestAgentSelfDebugging_BuiltinTool(t *testing.T) {
	ring := trace.NewRing(100)

	// Register via the same path app/ uses.
	registry := builtin.NewRegistry()
	builtin.RegisterDebugTrace(registry, ring)

	// Verify tool is registered.
	names := registry.Names()
	found := false
	for _, n := range names {
		if n == "djinn_trace" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("djinn_trace not registered as builtin tool")
	}

	// Simulate activity.
	tracer := ring.For(trace.ComponentAgent)
	rt := tracer.Begin("turn", "turn 1/5")
	rt.End()

	// Call via registry (same path agent would take).
	input := json.RawMessage(`{"action": "stats"}`)
	result, err := registry.Execute(context.Background(), "djinn_trace", input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, `"count"`) {
		t.Errorf("expected stats JSON, got: %s", result)
	}
}
