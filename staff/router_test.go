package staff

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/djinn/tools/builtin"
)

func testConfig() *StaffConfig {
	return &StaffConfig{
		Roles: []Role{
			{Name: "gensec", Mode: "plan", Slots: []string{"WorkTracker"}},
			{Name: "executor", Mode: "agent", Slots: []string{"WorkTracker", "CodeEditor", "Shell", "FileSearch"}},
		},
		Slots: []Slot{
			{Name: "WorkTracker", Backend: "scribe", Tools: []string{"artifact", "graph"}},
			{Name: "CodeEditor", Backend: "builtin", Tools: []string{"Read", "Write", "Edit"}},
			{Name: "Shell", Backend: "builtin", Tools: []string{"Bash"}},
			{Name: "FileSearch", Backend: "builtin", Tools: []string{"Glob", "Grep"}},
		},
	}
}

func TestSlotRouter_ExecutorSeesAllTools(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "executor")

	tools := router.All()
	// Executor should see: Read, Write, Edit, Bash, Glob, Grep (6 built-ins)
	if len(tools) < 6 {
		t.Fatalf("executor should see 6+ tools, got %d: %v", len(tools), router.Names())
	}
}

func TestSlotRouter_GenSecSeesNoCodeTools(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "gensec")

	tools := router.All()
	// GenSec only has WorkTracker slot → tools: artifact, graph
	// But those are MCP tools, not in the built-in registry.
	// So GenSec sees ZERO built-in tools.
	for _, tool := range tools {
		if tool.Name() == "Read" || tool.Name() == "Bash" || tool.Name() == "Edit" {
			t.Fatalf("gensec should NOT see %q", tool.Name())
		}
	}
}

func TestSlotRouter_ExecuteDeniedForWrongRole(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "gensec")

	// GenSec trying to call Read (CodeEditor slot) — should be denied.
	_, err := router.Execute(context.Background(), "Read", json.RawMessage(`{"file_path":"/etc/passwd"}`))
	if err == nil {
		t.Fatal("gensec should not be able to call Read")
	}
}

func TestSlotRouter_ExecuteAllowedForCorrectRole(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "executor")

	// Executor calling Read — should be ALLOWED (not permission denied).
	// The tool itself may return an error for the file path, but the
	// router should let it through. We check that the error is NOT
	// "not available for role", which would mean the router blocked it.
	_, err := router.Execute(context.Background(), "Read", json.RawMessage(`{"file_path":"go.mod"}`))
	// err may be non-nil (file not found etc.) but should NOT be "not available for role"
	if err != nil && err.Error() == `tool "Read" not available for role "executor"` {
		t.Fatal("executor should be allowed to call Read")
	}
}

func TestSlotRouter_RoleSwitchChangesVisibility(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "executor")

	// Executor sees built-in tools.
	if len(router.All()) < 6 {
		t.Fatal("executor should see 6+ tools")
	}

	// Switch to gensec.
	router.SetRole("gensec")

	// GenSec should see fewer tools.
	gensecTools := router.All()
	for _, tool := range gensecTools {
		if tool.Name() == "Bash" {
			t.Fatal("gensec should not see Bash after role switch")
		}
	}
}

func TestSlotRouter_UnknownRoleSeesNothing(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "nonexistent")

	if len(router.All()) != 0 {
		t.Fatalf("unknown role should see 0 tools, got %d", len(router.All()))
	}
}

func TestSlotRouter_MCPToolRouting(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	router := NewSlotRouter(cfg, registry, "gensec")

	// GenSec has WorkTracker slot with tools: artifact, graph
	// These would be MCP tools: mcp__scribe__artifact, mcp__scribe__graph
	// The router should allow the MCP-prefixed names.
	allowed := router.allowed
	if !allowed["mcp__scribe__artifact"] {
		t.Fatal("gensec should be allowed mcp__scribe__artifact via WorkTracker slot")
	}
	if !allowed["mcp__scribe__graph"] {
		t.Fatal("gensec should be allowed mcp__scribe__graph via WorkTracker slot")
	}
}
