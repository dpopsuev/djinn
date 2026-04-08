//go:build e2e

// e2e_llm_test.go — THE PoC test: can Djinn build an HTTP server?
//
// Requires: ANTHROPIC_API_KEY set in environment.
// Run: go test -tags e2e -run TestPoC_HTTPServer -v -timeout 300s ./arena/
package arena

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/contextmgr"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func TestPoC_HTTPServer(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set — skipping real LLM test")
	}

	// Real Claude driver — created once, reused by factory
	drv, err := claude.NewAPIDriver(driver.DriverConfig{
		Model: "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}

	// Actor factory — receives workspace, creates tools pointed at it
	factory := func(workspace string) (ActorFunc, error) {
		reg := builtin.NewRegistry()
		builtin.RegisterBuiltinTools(reg, workspace, workspace)
		sess := contextmgr.New("poc-test", "claude-sonnet-4", workspace)

		return func(ctx context.Context, prompt string) (string, error) {
			return agent.Run(ctx, agent.Config{
				Driver:       drv,
				Tools:        reg,
				Session:      sess,
				SystemPrompt: "You are a Go developer. Write files using the Write tool. Use Bash to compile. Work in the current directory.",
				MaxTurns:     15,
				ToolsEnabled: true,
				Mode:         agent.ModeAuto,
				Approve:      agent.AutoApprove,
				Enforcer:     policy.NopToolPolicyEnforcer{},
			}, prompt)
		}, nil
	}

	// Build the run
	runner, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewHTTPReferee()).
		WithActorFactory(factory).
		WithToolLevel(L0Mothballed).
		WithLogger(slog.Default()).
		Build()
	if err != nil {
		t.Fatalf("build runner: %v", err)
	}

	// Execute — the moment of truth
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	metrics, err := runner.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Structured output
	metricsJSON, _ := json.MarshalIndent(metrics, "", "  ")
	t.Logf("PoC RunMetrics:\n%s", string(metricsJSON))

	if !metrics.Pass {
		t.Fatal("PoC FAILED — Djinn cannot build an HTTP server")
	}

	t.Log("PoC PASSED — Djinn built a working HTTP server!")
}
