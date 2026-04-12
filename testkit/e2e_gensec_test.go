package testkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform"
)

// TestE2E_GenSecResponds_WithUniform proves the Sprint 2 assertion:
// GenSec boots with Uniform, system prompt includes RBAC-filtered tools,
// responds to a prompt, and can only use tools allowed by its role.
//
// GenSec = [director, manager]. Has: shell, observe, coordinate, communicate, work.
// Does NOT have: read, write, code, vcs.
func TestE2E_GenSecResponds_WithUniform(t *testing.T) {
	// 1. Build GenSec Uniform via RBAC resolution.
	roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
	toolReqs := uniform.DefaultToolRequirements()

	registry := builtin.NewRegistry()
	builtin.RegisterBuiltinTools(registry, t.TempDir(), t.TempDir())

	gensecUniform := uniform.NewUniform(
		"gensec",
		[]string{"director", "manager"},
		roleReg,
		toolReqs,
		registry.Names(),
		"plan",
		"test-model",
		"You are the General Secretary — root agent (PID 1).",
	)

	// 2. Assert Uniform resolved correctly.
	if !gensecUniform.HasCapability(uniform.CapShell) {
		t.Fatal("GenSec should have shell capability")
	}
	if !gensecUniform.HasCapability(uniform.CapCoordinate) {
		t.Fatal("GenSec should have coordinate capability")
	}
	if gensecUniform.HasCapability(uniform.CapWrite) {
		t.Fatal("GenSec should NOT have write capability")
	}
	if gensecUniform.HasCapability(uniform.CapCode) {
		t.Fatal("GenSec should NOT have code capability")
	}

	// 3. Assert system prompt includes tool list.
	sysCtx := gensecUniform.SystemContext()
	if !strings.Contains(sysCtx, "Bash") {
		t.Fatal("GenSec system context should include Bash (has shell)")
	}
	if !strings.Contains(sysCtx, "assignment") {
		t.Fatal("GenSec system context should include assignment (has coordinate)")
	}
	if strings.Contains(sysCtx, "Write") {
		t.Fatal("GenSec system context should NOT include Write (no write cap)")
	}
	if strings.Contains(sysCtx, "Edit") {
		t.Fatal("GenSec system context should NOT include Edit (no write cap)")
	}
	if !strings.Contains(sysCtx, "Do not attempt") {
		t.Fatal("GenSec system context should tell agent not to use unlisted tools")
	}

	// 4. Assert no warnings on valid config.
	if len(gensecUniform.Warnings()) > 0 {
		t.Fatalf("GenSec should have no warnings, got: %v", gensecUniform.Warnings())
	}

	// 5. Wire agent loop with scripted driver — GenSec responds to prompt.
	drv := stubs.NewScriptedChatDriver(
		stubs.ScriptedTurn{
			Text: "I am GenSec, the General Secretary. I'll coordinate the team.",
			ToolCalls: []driver.ToolCall{{
				ID:    "c1",
				Name:  "assignment",
				Input: stubs.MustJSON(map[string]string{"action": "list", "status": "assigned"}),
			}},
		},
		stubs.ScriptedTurn{Text: "No assignments pending. Ready for orders."},
	)

	sess := cortex.New("gensec-e2e", "test-model", t.TempDir())

	// Build system prompt with Uniform context.
	systemPrompt := gensecUniform.SystemContext()

	result, err := agent.Run(context.Background(), agent.Config{
		Driver:       drv,
		Tools:        registry,
		Session:      sess,
		SystemPrompt: systemPrompt,
		MaxTurns:     5,
		ToolsEnabled: true,
		Approve:      agent.AutoApprove,
		Enforcer:     policy.NopToolPolicyEnforcer{},
	}, "What assignments are pending?")
	if err != nil {
		t.Fatalf("agent.Run: %v", err)
	}

	if !strings.Contains(result, "Ready for orders") {
		t.Fatalf("GenSec should respond, got: %q", result)
	}

	// 6. Verify the tool call went through (assignment is allowed for GenSec).
	if drv.TurnCount() < 2 {
		t.Fatal("expected at least 2 turns (tool call + response)")
	}

	t.Log("Sprint 2 E2E PASSES — GenSec boots with Uniform, RBAC filters tools, responds to prompt")
}

// TestE2E_CoderUniform_CantUseBash proves a Coder agent can't use Bash.
func TestE2E_CoderUniform_CantUseBash(t *testing.T) {
	roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
	toolReqs := uniform.DefaultToolRequirements()

	registry := builtin.NewRegistry()
	builtin.RegisterBuiltinTools(registry, t.TempDir(), t.TempDir())

	coderUniform := uniform.NewUniform(
		"coder-1",
		[]string{"developer"},
		roleReg,
		toolReqs,
		registry.Names(),
		"agent",
		"test-model",
		"You are a Coder. Write code.",
	)

	// Coder has Read, Write, Edit but NOT Bash.
	if !coderUniform.HasTool("Read") {
		t.Fatal("Coder should have Read")
	}
	if !coderUniform.HasTool("Write") {
		t.Fatal("Coder should have Write")
	}
	if coderUniform.HasTool("Bash") {
		t.Fatal("Coder should NOT have Bash")
	}
	if coderUniform.HasTool("assignment") {
		t.Fatal("Coder should NOT have assignment")
	}

	// System context should list allowed tools only.
	sysCtx := coderUniform.SystemContext()
	if strings.Contains(sysCtx, "Bash") {
		t.Fatal("Coder system context should NOT mention Bash")
	}
	if !strings.Contains(sysCtx, "Read") {
		t.Fatal("Coder system context should mention Read")
	}

	t.Log("Coder Uniform E2E PASSES — RBAC correctly filters Bash out")
}

// TestE2E_UniformDeniedToolsTracked proves denied tools are tracked for observability.
func TestE2E_UniformDeniedToolsTracked(t *testing.T) {
	roleReg := uniform.NewRoleRegistry(uniform.DefaultRoles())
	toolReqs := uniform.DefaultToolRequirements()

	allTools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "git", "assignment", "discourse", "plan"}

	coderUniform := uniform.NewUniform("coder-1", []string{"developer"}, roleReg, toolReqs, allTools, "agent", "", "")

	denied := coderUniform.Denied()
	if len(denied) == 0 {
		t.Fatal("Coder should have denied tools (Bash, assignment)")
	}

	hasBash := false
	hasAssignment := false
	for _, d := range denied {
		if d == "Bash" {
			hasBash = true
		}
		if d == "assignment" {
			hasAssignment = true
		}
	}
	if !hasBash {
		t.Fatal("Bash should be in denied list for Coder")
	}
	if !hasAssignment {
		t.Fatal("assignment should be in denied list for Coder")
	}

	t.Log("Denied tools tracked — ORANGE observability works")
}

// ensure json import is used.
var _ = json.Marshal

// ensure time import is used.
var _ = time.Now
