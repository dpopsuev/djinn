package substrate_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/battery/tool"
	"github.com/dpopsuev/djinn/arena"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/troupe/testkit"
)

// echoExecutor returns the tool name as output.
type echoExecutor struct{}

func (echoExecutor) Execute(_ context.Context, name string, _ json.RawMessage) (string, error) {
	return "executed: " + name, nil
}
func (echoExecutor) All() []tool.Tool { return nil }
func (echoExecutor) Names() []string  { return []string{"Read", "Write", "Bash"} }

// TestE2E_Skeleton proves the full pipeline wires with stubs:
// MockOperator → StubSubstrate → tools → StubReferee → RunMetrics
//
// No real LLM, no real filesystem, no real network.
// This is the forge — proves composition before any real backend exists.
func TestE2E_Skeleton(t *testing.T) {
	ctx := context.Background()

	// 1. Build the Substrate with echo tools + security gate
	exec := echoExecutor{}
	gate := middleware.AsSecurityGate(alwaysAllowGate{})
	env, err := middleware.NewBuilder(exec).WithGate(gate).Build()
	if err != nil {
		t.Fatalf("build envelope: %v", err)
	}

	sub := substrate.NewStubSubstrate(exec, testkit.NewStubEventLog())
	sub.SetEnvelope(env)

	// 2. Define the scenario + referee
	scenario := arena.NewStubScenario("http-service", "Build an HTTP server that returns JSON on GET /health")
	referee := arena.NewStubReferee(arena.CheckResult{Pass: true, Score: 0.9})

	// 3. Create a mock operator with canned prompts
	operator := arena.NewMockOperator(
		"Create a Go HTTP server with a /health endpoint that returns {\"status\": \"ok\"}",
	)

	// 4. Simulate the run: operator → substrate → tools → referee
	start := time.Now()

	// Operator provides the prompt
	prompt, err := operator.Perform(ctx, "ready to start")
	if err != nil {
		t.Fatalf("operator: %v", err)
	}
	if prompt == "" {
		t.Fatal("operator should provide a prompt")
	}

	// Substrate spawns an agent
	agentID, err := sub.Spawn(ctx, substrate.SpawnConfig{
		Role:  "executor",
		Model: "haiku",
		Tools: []string{"Read", "Write", "Bash"},
	})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	// Agent executes tools through the Envelope
	result, err := env.Execute(ctx, "Write", json.RawMessage(`{"path":"main.go","content":"package main"}`))
	if err != nil {
		t.Fatalf("tool execute: %v", err)
	}

	// Substrate observes the tool call
	sub.Observe(ctx, substrate.Observation{
		AgentID:  agentID,
		Tool:     "Write",
		Duration: 50,
	})

	// Referee checks the result
	check, err := referee.Check(ctx, scenario.ID(), "/tmp/project")
	if err != nil {
		t.Fatalf("referee: %v", err)
	}

	// Substrate kills the agent
	if err := sub.Kill(ctx, agentID); err != nil {
		t.Fatalf("kill: %v", err)
	}

	elapsed := time.Since(start)

	// 5. Record metrics
	metrics := arena.RunMetrics{
		ScenarioID:  scenario.ID(),
		ToolLevel:   arena.L0Mothballed,
		TimeElapsed: elapsed,
		TokensIn:    0,
		TokensOut:   0,
		Pass:        check.Pass,
		Score:       check.Score,
	}

	// 6. Assert the full pipeline wired correctly
	if !metrics.Pass {
		t.Fatal("expected pass")
	}
	if metrics.Score != 0.9 {
		t.Fatalf("score = %f, want 0.9", metrics.Score)
	}
	if len(sub.Observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(sub.Observations))
	}
	if sub.Observations[0].Tool != "Write" {
		t.Fatalf("observed tool = %q, want Write", sub.Observations[0].Tool)
	}
	if len(sub.Spawned) != 1 {
		t.Fatalf("spawned = %d, want 1", len(sub.Spawned))
	}
	if len(sub.Killed) != 1 {
		t.Fatalf("killed = %d, want 1", len(sub.Killed))
	}
	if operator.Calls != 1 {
		t.Fatalf("operator calls = %d, want 1", operator.Calls)
	}
	if referee.Checks != 1 {
		t.Fatalf("referee checks = %d, want 1", referee.Checks)
	}
	if result == "" {
		t.Fatal("tool result should not be empty")
	}

	t.Logf("E2E skeleton passed: scenario=%s, score=%.1f, time=%v",
		metrics.ScenarioID, metrics.Score, metrics.TimeElapsed)
}

// alwaysAllowGate satisfies middleware.Gate for the skeleton.
type alwaysAllowGate struct{}

func (alwaysAllowGate) Check(_ context.Context, _ string, _ json.RawMessage) (middleware.Verdict, error) {
	return middleware.Verdict{Allowed: true}, nil
}
