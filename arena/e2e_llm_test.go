//go:build e2e

// e2e_llm_test.go — THE PoC test: can Djinn build an HTTP server?
//
// Requires: ANTHROPIC_API_KEY set in environment.
// Run: go test -tags e2e -run TestPoC_HTTPServer -v -timeout 300s ./arena/
package arena

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/contextmgr"
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/troupe/execution"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

func TestPoC_HTTPServer(t *testing.T) {
	if os.Getenv("DJINN_PROVIDER") == "" {
		t.Skip("DJINN_PROVIDER not set — skipping real LLM test")
	}

	// Real provider via Troupe — provider-agnostic
	provider, err := execution.NewProviderFromEnv("DJINN_PROVIDER")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	model := os.Getenv("DJINN_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	// Actor factory — receives workspace, creates tools pointed at it
	factory := func(workspace string) (ActorFunc, error) {
		reg := builtin.NewRegistry()
		builtin.RegisterBuiltinTools(reg, workspace, workspace)
		sess := contextmgr.New("poc-test", model, workspace)

		// Convert tools for the driver
		tools := registryToTools(reg)

		systemPrompt := fmt.Sprintf(
			"You are a Go developer. Write files using the Write tool. "+
				"Use Bash ONLY to compile (go build), NEVER to run the server. "+
				"All file paths MUST be absolute, rooted at %s. "+
				"Example: %s/main.go. "+
				"After writing all files, compile with: cd %s && go build -o server .",
			workspace, workspace, workspace,
		)

		return func(ctx context.Context, prompt string) (string, error) {
			drv := troupedriver.New(provider, model,
				troupedriver.WithTools(tools),
				troupedriver.WithSystemPrompt(systemPrompt),
			)

			if err := drv.Start(ctx, ""); err != nil {
				return "", fmt.Errorf("start driver: %w", err)
			}
			defer drv.Stop(ctx) //nolint:errcheck // best-effort

			return agent.Run(ctx, agent.Config{
				Driver:       drv,
				Tools:        reg,
				Session:      sess,
				SystemPrompt: systemPrompt,
				MaxTurns:     10,
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

// registryToTools converts builtin.Registry to anyllm.Tool definitions.
func registryToTools(reg *builtin.Registry) []anyllm.Tool {
	all := reg.All()
	tools := make([]anyllm.Tool, 0, len(all))
	for _, t := range all {
		var params map[string]any
		_ = json.Unmarshal(t.InputSchema(), &params)
		tools = append(tools, anyllm.Tool{
			Type: "function",
			Function: anyllm.Function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  params,
			},
		})
	}
	return tools
}
