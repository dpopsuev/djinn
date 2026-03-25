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
			{Name: "gensec", Mode: "plan", ToolCapabilities: []string{"WorkTracking"}},
			{Name: "executor", Mode: "agent", ToolCapabilities: []string{"WorkTracking", "FileEditing", "ShellExecution", "FileSearching"}},
		},
		ToolCapabilities: []ToolCapability{
			{Name: "WorkTracking", Backend: "scribe", Tools: []string{"artifact", "graph"}},
			{Name: "FileEditing", Backend: "builtin", Tools: []string{"Read", "Write", "Edit"}},
			{Name: "ShellExecution", Backend: "builtin", Tools: []string{"Bash"}},
			{Name: "FileSearching", Backend: "builtin", Tools: []string{"Glob", "Grep"}},
		},
	}
}

func TestToolClearance_ExecutorSeesAllTools(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "executor")

	tools := clearance.All()
	if len(tools) < 6 {
		t.Fatalf("executor should see 6+ tools, got %d: %v", len(tools), clearance.Names())
	}
}

func TestToolClearance_GenSecSeesNoCodeTools(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "gensec")

	tools := clearance.All()
	for _, tool := range tools {
		if tool.Name() == "Read" || tool.Name() == "Bash" || tool.Name() == "Edit" {
			t.Fatalf("gensec should NOT see %q", tool.Name())
		}
	}
}

func TestToolClearance_ExecuteDeniedForWrongRole(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "gensec")

	_, err := clearance.Execute(context.Background(), "Read", json.RawMessage(`{"file_path":"/etc/passwd"}`))
	if err == nil {
		t.Fatal("gensec should not be able to call Read")
	}
}

func TestToolClearance_ExecuteAllowedForCorrectRole(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "executor")

	_, err := clearance.Execute(context.Background(), "Read", json.RawMessage(`{"file_path":"go.mod"}`))
	if err != nil && err.Error() == `tool "Read" not available for role "executor"` {
		t.Fatal("executor should be allowed to call Read")
	}
}

func TestToolClearance_RoleSwitchChangesVisibility(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "executor")

	if len(clearance.All()) < 6 {
		t.Fatal("executor should see 6+ tools")
	}

	clearance.SetRole("gensec")

	gensecTools := clearance.All()
	for _, tool := range gensecTools {
		if tool.Name() == "Bash" {
			t.Fatal("gensec should not see Bash after role switch")
		}
	}
}

func TestToolClearance_UnknownRoleSeesNothing(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "nonexistent")

	if len(clearance.All()) != 0 {
		t.Fatalf("unknown role should see 0 tools, got %d", len(clearance.All()))
	}
}

func TestToolClearance_MCPToolRouting(t *testing.T) {
	cfg := testConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "gensec")

	allowed := clearance.allowed
	if !allowed["mcp__scribe__artifact"] {
		t.Fatal("gensec should be allowed mcp__scribe__artifact via WorkTracking capability")
	}
	if !allowed["mcp__scribe__graph"] {
		t.Fatal("gensec should be allowed mcp__scribe__graph via WorkTracking capability")
	}
}
