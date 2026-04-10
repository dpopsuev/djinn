package crucible

import (
	"context"
	"testing"
	"time"
)

func TestStubReferee_ReturnsConfiguredResult(t *testing.T) {
	ref := NewStubReferee(CheckResult{Pass: true, Score: 0.95})

	result, err := ref.Check(context.Background(), "http-service", "/tmp/project")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		t.Fatal("expected pass")
	}
	if result.Score != 0.95 {
		t.Fatalf("score = %f, want 0.95", result.Score)
	}
	if ref.Checks != 1 {
		t.Fatalf("checks = %d, want 1", ref.Checks)
	}
}

func TestStubReferee_Fail(t *testing.T) {
	ref := NewStubReferee(CheckResult{Pass: false, Errors: []string{"build failed"}})

	result, _ := ref.Check(context.Background(), "http-service", "/tmp/project")
	if result.Pass {
		t.Fatal("expected fail")
	}
	if len(result.Errors) != 1 || result.Errors[0] != "build failed" {
		t.Fatalf("errors = %v", result.Errors)
	}
}

func TestStubScenario_ReturnsSpec(t *testing.T) {
	s := NewStubScenario("http-service", "Build an HTTP server that returns JSON on GET /health")

	if s.ID() != "http-service" {
		t.Fatalf("ID = %q", s.ID())
	}
	if s.Spec() == "" {
		t.Fatal("spec is empty")
	}
	if s.Timeout() != 10*time.Minute {
		t.Fatalf("timeout = %v", s.Timeout())
	}
	b := s.Budget()
	if b.MaxTokens != 100000 {
		t.Fatalf("budget.MaxTokens = %d", b.MaxTokens)
	}
}

func TestMockOperator_FeedsPromptsSequentially(t *testing.T) {
	op := NewMockOperator("build the server", "add tests", "deploy")

	ctx := context.Background()
	r1, _ := op.Perform(ctx, "what should I do?")
	r2, _ := op.Perform(ctx, "done, what next?")
	r3, _ := op.Perform(ctx, "done, what next?")
	r4, _ := op.Perform(ctx, "anything else?")

	if r1 != "build the server" {
		t.Fatalf("r1 = %q", r1)
	}
	if r2 != "add tests" {
		t.Fatalf("r2 = %q", r2)
	}
	if r3 != "deploy" {
		t.Fatalf("r3 = %q", r3)
	}
	if r4 != "" {
		t.Fatalf("r4 = %q, want empty (exhausted)", r4)
	}
	if op.Calls != 4 {
		t.Fatalf("calls = %d, want 4", op.Calls)
	}
}

// E2E skeleton: proves the harness wires before any real scenario exists.
func TestArena_E2E_Skeleton(t *testing.T) {
	ctx := context.Background()

	// Setup
	scenario := NewStubScenario("http-service", "Build an HTTP server")
	referee := NewStubReferee(CheckResult{Pass: true, Score: 0.85})
	operator := NewMockOperator("build it")

	// Simulate a run: operator provides prompt, "agent" produces project, referee checks
	start := time.Now()

	prompt, err := operator.Perform(ctx, "ready")
	if err != nil {
		t.Fatal(err)
	}
	if prompt == "" {
		t.Fatal("operator should provide a prompt")
	}

	// "Agent" would work here — for skeleton, we skip straight to referee
	result, err := referee.Check(ctx, scenario.ID(), "/tmp/fake-project")
	if err != nil {
		t.Fatal(err)
	}

	elapsed := time.Since(start)

	// Record metrics
	metrics := RunMetrics{
		ScenarioID:  scenario.ID(),
		ToolLevel:   L0Mothballed,
		TimeElapsed: elapsed,
		TokensIn:    0, // stub — no real agent
		TokensOut:   0,
		Pass:        result.Pass,
		Score:       result.Score,
	}

	// Assert the harness wired correctly
	if metrics.ScenarioID != "http-service" {
		t.Fatalf("scenario = %q", metrics.ScenarioID)
	}
	if !metrics.Pass {
		t.Fatal("expected pass from stub referee")
	}
	if metrics.Score != 0.85 {
		t.Fatalf("score = %f, want 0.85", metrics.Score)
	}
	if metrics.ToolLevel != L0Mothballed {
		t.Fatalf("tool level = %q, want L0", metrics.ToolLevel)
	}
	if referee.Checks != 1 {
		t.Fatal("referee should have been called once")
	}
	if operator.Calls != 1 {
		t.Fatal("operator should have been called once")
	}
}
