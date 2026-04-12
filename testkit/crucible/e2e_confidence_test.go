//go:build e2e

// e2e_confidence_test.go — Real LLM confidence tests.
// Uses the full Restructure stack: RBAC Uniform + real tools + real LLM.
// Gated by DJINN_PROVIDER env var. Skip if not set.
//
// Run: go test -tags e2e -run TestConfidence -v -timeout 120s ./testkit/crucible/
package crucible

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/testkit"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform"
	"github.com/dpopsuev/troupe/execution"
)

func skipIfNoProvider(t *testing.T) {
	t.Helper()
	if os.Getenv("DJINN_PROVIDER") == "" {
		t.Skip("DJINN_PROVIDER not set — skipping real LLM test")
	}
}

func realDriver(t *testing.T, registry *builtin.Registry) *troupedriver.ChatDriver {
	t.Helper()
	provider, err := execution.NewProviderFromEnv("DJINN_PROVIDER")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := os.Getenv("DJINN_MODEL")
	if model == "" {
		t.Fatal("DJINN_MODEL not set — required for real LLM tests")
	}
	tools := registryToTools(registry)
	return troupedriver.New(provider, model, troupedriver.WithTools(tools))
}

// TestConfidence_AgentWritesFile proves the full stack with a real LLM:
// RBAC Uniform (Coder role) + real tools (Write) + real LLM → file on disk.
// Uses Mirage workspace for isolation — no stale files, no wrong paths.
func TestConfidence_AgentWritesFile(t *testing.T) {
	skipIfNoProvider(t)

	testkit.AgentTest(t, func(ws *testkit.TestWorkspace) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		workDir := ws.Dir()

		roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
		toolReqs := uniform.DefaultToolRequirements()
		registry := builtin.NewRegistry()
		builtin.RegisterBuiltinTools(registry, workDir, workDir)

		coderUniform := uniform.NewUniform(
			"coder-1", []string{"developer"},
			roleReg, toolReqs, registry.Names(),
			"agent", "",
			fmt.Sprintf("You are a Coder. Use the Write tool to create files. ALL file paths MUST be absolute, rooted at %s. Example: %s/main.go", workDir, workDir),
		)

		drv := realDriver(t, registry)
		if err := drv.Start(ctx, coderUniform.SystemContext()); err != nil {
			t.Fatalf("start driver: %v", err)
		}
		defer drv.Stop(ctx) //nolint:errcheck // best-effort cleanup

		sess := cortex.New("confidence-write", os.Getenv("DJINN_PROVIDER"), workDir)

		result, err := agent.Run(ctx, agent.Config{
			Driver:       drv,
			Tools:        registry,
			Session:      sess,
			SystemPrompt: coderUniform.SystemContext(),
			MaxTurns:     10,
			ToolsEnabled: true,
			Approve:      agent.AutoApprove,
			Enforcer:     policy.NopToolPolicyEnforcer{},
		}, "Create a file called hello.go with package main and a main function that prints 'hello djinn'")
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		t.Logf("Agent response: %s", result)

		if !ws.HasFile("hello.go") {
			t.Fatal("file not created in workspace")
		}

		content, err := ws.ReadFile("hello.go")
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if !strings.Contains(content, "package main") {
			t.Fatalf("file missing 'package main': %s", content)
		}
		if !strings.Contains(content, "func main") {
			t.Fatalf("file missing 'func main': %s", content)
		}

		t.Log("CONFIDENCE PASSES — real LLM + RBAC Uniform + Mirage workspace → file on disk")
	})
}

// TestConfidence_AgentReadsAndEdits proves read→edit cycle with real LLM.
// Uses Mirage workspace for isolation.
func TestConfidence_AgentReadsAndEdits(t *testing.T) {
	skipIfNoProvider(t)

	testkit.AgentTest(t, func(ws *testkit.TestWorkspace) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		workDir := ws.Dir()

		// Seed a file into the workspace.
		ws.WriteFile(t, "config.go", "package config\n\nvar Version = \"0.0.0\"\n")

		roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
		toolReqs := uniform.DefaultToolRequirements()
		registry := builtin.NewRegistry()
		builtin.RegisterBuiltinTools(registry, workDir, workDir)

		coderUniform := uniform.NewUniform(
			"coder-1", []string{"developer"},
			roleReg, toolReqs, registry.Names(),
			"agent", "",
			fmt.Sprintf("You are a Coder. Use Read and Edit tools. ALL file paths MUST be absolute, rooted at %s.", workDir),
		)

		drv := realDriver(t, registry)
		if err := drv.Start(ctx, coderUniform.SystemContext()); err != nil {
			t.Fatalf("start driver: %v", err)
		}
		defer drv.Stop(ctx) //nolint:errcheck // best-effort cleanup

		sess := cortex.New("confidence-edit", os.Getenv("DJINN_PROVIDER"), workDir)

		result, err := agent.Run(ctx, agent.Config{
			Driver:       drv,
			Tools:        registry,
			Session:      sess,
			SystemPrompt: coderUniform.SystemContext(),
			MaxTurns:     10,
			ToolsEnabled: true,
			Approve:      agent.AutoApprove,
			Enforcer:     policy.NopToolPolicyEnforcer{},
		}, fmt.Sprintf("Read %s/config.go and change the Version from 0.0.0 to 1.0.0", workDir))
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		t.Logf("Agent response: %s", result)

		content, err := ws.ReadFile("config.go")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(content, "1.0.0") {
			t.Fatalf("edit failed — file still has old version: %s", content)
		}
		if strings.Contains(content, "0.0.0") {
			t.Fatalf("edit failed — old version still present: %s", content)
		}

		t.Log("CONFIDENCE PASSES — real LLM + Read + Edit + Mirage workspace → file modified correctly")
	})
}
