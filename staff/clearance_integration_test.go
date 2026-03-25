package staff

import (
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// TestToolClearance_PolicyEnforcer_Integration verifies the full stack:
// StaffConfig → ToolClearance (visibility) + ToolPolicyEnforcer (enforcement).
// Role switch changes both which tools the agent sees AND which tools
// the policy enforcer allows.
func TestToolClearance_PolicyEnforcer_Integration(t *testing.T) {
	cfg := DefaultConfig()
	registry := builtin.NewRegistry()
	clearance := NewToolClearance(cfg, registry, "executor")
	enforcer := policy.NewDefaultToolPolicyEnforcer()

	// Build token from executor's resolved tool names (not capability names).
	execRole := cfg.RoleMap()["executor"]
	execToken := policy.CapabilityToken{
		AllowedTools: cfg.ResolveToolNames(execRole.ToolCapabilities),
	}

	// Executor can call Read (via FileEditing capability).
	if err := enforcer.Check(execToken, "Read", nil); err != nil {
		t.Fatalf("executor should be allowed Read: %v", err)
	}
	// Executor can call Bash (via ShellExecution capability).
	if err := enforcer.Check(execToken, "Bash", nil); err != nil {
		t.Fatalf("executor should be allowed Bash: %v", err)
	}
	// Executor can call Glob (via FileSearching capability).
	if err := enforcer.Check(execToken, "Glob", nil); err != nil {
		t.Fatalf("executor should be allowed Glob: %v", err)
	}

	// ToolClearance also shows the right tools.
	execTools := clearance.Names()
	found := map[string]bool{}
	for _, name := range execTools {
		found[name] = true
	}
	if !found["Read"] || !found["Bash"] || !found["Glob"] {
		t.Fatalf("executor clearance should show Read/Bash/Glob, got: %v", execTools)
	}

	// Switch to gensec.
	clearance.SetRole("gensec")
	gensecRole := cfg.RoleMap()["gensec"]
	gensecToken := policy.CapabilityToken{
		AllowedTools: cfg.ResolveToolNames(gensecRole.ToolCapabilities),
	}

	// GenSec CANNOT call Read (no FileEditing capability).
	err := enforcer.Check(gensecToken, "Read", nil)
	if err == nil {
		t.Fatal("gensec should NOT be allowed Read")
	}
	if !errors.Is(err, policy.ErrDeniedTool) {
		t.Fatalf("err = %v, want ErrDeniedTool", err)
	}

	// GenSec CANNOT call Bash (no ShellExecution capability).
	err = enforcer.Check(gensecToken, "Bash", nil)
	if err == nil {
		t.Fatal("gensec should NOT be allowed Bash")
	}

	// GenSec CAN call MCP-prefixed scribe tools (via WorkTracking).
	if err := enforcer.Check(gensecToken, "mcp__scribe__artifact", nil); err != nil {
		t.Fatalf("gensec should be allowed mcp__scribe__artifact: %v", err)
	}

	// ToolClearance also hides code tools from gensec.
	gensecTools := clearance.Names()
	for _, name := range gensecTools {
		if name == "Read" || name == "Bash" || name == "Edit" {
			t.Fatalf("gensec clearance should NOT show %q", name)
		}
	}
}

// TestToolClearance_PolicyEnforcer_AuditorRole verifies auditor
// has WorkTracking + RuleResolution but NOT FileEditing or ShellExecution.
func TestToolClearance_PolicyEnforcer_AuditorRole(t *testing.T) {
	cfg := DefaultConfig()
	enforcer := policy.NewDefaultToolPolicyEnforcer()

	audRole := cfg.RoleMap()["auditor"]
	audToken := policy.CapabilityToken{
		AllowedTools: cfg.ResolveToolNames(audRole.ToolCapabilities),
	}

	// Auditor can call lexicon (via RuleResolution).
	if err := enforcer.Check(audToken, "mcp__lex__lexicon", nil); err != nil {
		t.Fatalf("auditor should be allowed lexicon: %v", err)
	}

	// Auditor cannot call Bash.
	if err := enforcer.Check(audToken, "Bash", nil); err == nil {
		t.Fatal("auditor should NOT be allowed Bash")
	}
}

// TestResolveToolNames_NotCapabilityNames verifies the bug fix:
// ResolveToolNames returns tool names like "Read", NOT capability names like "FileEditing".
func TestResolveToolNames_NotCapabilityNames(t *testing.T) {
	cfg := DefaultConfig()
	execRole := cfg.RoleMap()["executor"]
	tools := cfg.ResolveToolNames(execRole.ToolCapabilities)

	// Should contain tool names.
	toolSet := map[string]bool{}
	for _, name := range tools {
		toolSet[name] = true
	}

	if !toolSet["Read"] {
		t.Error("should contain tool name 'Read'")
	}
	if !toolSet["Bash"] {
		t.Error("should contain tool name 'Bash'")
	}

	// Should NOT contain capability names.
	if toolSet["FileEditing"] {
		t.Error("should NOT contain capability name 'FileEditing'")
	}
	if toolSet["ShellExecution"] {
		t.Error("should NOT contain capability name 'ShellExecution'")
	}
}
