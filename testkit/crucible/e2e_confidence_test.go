//go:build e2e

// e2e_confidence_test.go — Real LLM confidence tests.
// Uses the full Restructure stack: RBAC Uniform + real tools + real LLM.
// Gated by DJINN_PROVIDER env var. Skip if not set.
//
// Run: go test -tags e2e -run TestConfidence -v -timeout 120s ./testkit/crucible/
package crucible

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/policy"
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

func realDriver(t *testing.T) *troupedriver.ChatDriver {
	t.Helper()
	provider, err := execution.NewProviderFromEnv("DJINN_PROVIDER")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := os.Getenv("DJINN_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return troupedriver.New(provider, model)
}

// TestConfidence_AgentWritesFile proves the full stack with a real LLM:
// RBAC Uniform (Coder role) + real tools (Write) + real LLM → file on disk.
func TestConfidence_AgentWritesFile(t *testing.T) {
	skipIfNoProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	workDir := t.TempDir()

	// RBAC: Coder Uniform (has write, no shell).
	roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
	toolReqs := uniform.DefaultToolRequirements()
	registry := builtin.NewRegistry()
	builtin.RegisterBuiltinTools(registry, workDir, workDir)

	coderUniform := uniform.NewUniform(
		"coder-1", []string{"developer"},
		roleReg, toolReqs, registry.Names(),
		"agent", "", "You are a Coder. Write code. Use the Write tool to create files.",
	)

	// Real LLM driver.
	drv := realDriver(t)
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

	// Verify file exists.
	filePath := filepath.Join(workDir, "hello.go")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "package main") {
		t.Fatalf("file missing 'package main': %s", content)
	}
	if !strings.Contains(content, "func main") {
		t.Fatalf("file missing 'func main': %s", content)
	}

	t.Log("CONFIDENCE PASSES — real LLM + RBAC Uniform + real Write tool → file on disk")
}

// TestConfidence_AgentReadsAndEdits proves read→edit cycle with real LLM.
func TestConfidence_AgentReadsAndEdits(t *testing.T) {
	skipIfNoProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	workDir := t.TempDir()

	// Seed a file.
	seedPath := filepath.Join(workDir, "config.go")
	os.WriteFile(seedPath, []byte("package config\n\nvar Version = \"0.0.0\"\n"), 0o644) //nolint:errcheck // best-effort cleanup

	roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
	toolReqs := uniform.DefaultToolRequirements()
	registry := builtin.NewRegistry()
	builtin.RegisterBuiltinTools(registry, workDir, workDir)

	coderUniform := uniform.NewUniform(
		"coder-1", []string{"developer"},
		roleReg, toolReqs, registry.Names(),
		"agent", "", "You are a Coder. Read and edit files using the Read and Edit tools.",
	)

	drv := realDriver(t)
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
	}, "Read config.go and change the Version from 0.0.0 to 1.0.0")
	if err != nil {
		t.Fatalf("agent.Run: %v", err)
	}

	t.Logf("Agent response: %s", result)

	// Verify edit.
	data, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "1.0.0") {
		t.Fatalf("edit failed — file still has old version: %s", data)
	}
	if strings.Contains(string(data), "0.0.0") {
		t.Fatalf("edit failed — old version still present: %s", data)
	}

	t.Log("CONFIDENCE PASSES — real LLM + Read + Edit → file modified correctly")
}
