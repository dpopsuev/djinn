package acceptance

import (
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// TestE2E_StaffingPipeline verifies the role pipeline through the Model:
//   - Default role is gensec
//   - /role executor switches to executor with FileEditing tools
//   - Auditor does NOT have FileEditing tools
//   - GenSec has WorkTracking tools
func TestE2E_StaffingPipeline(t *testing.T) {
	sess := session.New("staff-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Driver:  &stubs.StubChatDriver{},
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := toModelPtr(m2)

	// Default role is gensec.
	view := model.View()
	if !strings.Contains(strings.ToUpper(view), "GENSEC") {
		t.Fatal("default role should be gensec")
	}

	// GenSec capabilities: WorkTracking is in gensec's tool_capabilities.
	cfg := staff.DefaultConfig()
	gensec := cfg.RoleMap()["gensec"]
	hasWorkTracking := false
	for _, cap := range gensec.ToolCapabilities {
		if cap == "WorkTracking" {
			hasWorkTracking = true
		}
	}
	if !hasWorkTracking {
		t.Fatal("gensec should have WorkTracking capability")
	}

	// Switch to executor via /role command.
	model.SetTextInput("/role executor")
	m3 := multiUpdate(t, *model, tea.KeyMsg{Type: tea.KeyEnter})
	model2 := toModelPtr(m3)

	view2 := model2.View()
	if !strings.Contains(strings.ToUpper(view2), "EXECUTOR") {
		t.Fatal("dashboard should show EXECUTOR after role switch")
	}

	// Executor should have FileEditing capability.
	executor := cfg.RoleMap()["executor"]
	hasFileEditing := false
	for _, cap := range executor.ToolCapabilities {
		if cap == "FileEditing" {
			hasFileEditing = true
		}
	}
	if !hasFileEditing {
		t.Fatal("executor should have FileEditing capability")
	}

	// Auditor should NOT have FileEditing.
	auditor := cfg.RoleMap()["auditor"]
	for _, cap := range auditor.ToolCapabilities {
		if cap == "FileEditing" {
			t.Fatal("auditor should NOT have FileEditing capability")
		}
	}

	// Switch to auditor and verify no FileEditing.
	model2.SetTextInput("/role auditor")
	m4 := multiUpdate(t, *model2, tea.KeyMsg{Type: tea.KeyEnter})
	model3 := toModelPtr(m4)

	view3 := model3.View()
	if !strings.Contains(strings.ToUpper(view3), "AUDITOR") {
		t.Fatal("dashboard should show AUDITOR after role switch")
	}
}

// TestE2E_RoleCapabilityResolution verifies that ResolveToolNames converts
// capability names to actual tool names. The executor role should resolve
// to concrete tools like Read, Write, Edit, Bash, Glob, Grep — not
// abstract capability names like FileEditing or ShellExecution.
func TestE2E_RoleCapabilityResolution(t *testing.T) {
	cfg := staff.DefaultConfig()
	executor := cfg.RoleMap()["executor"]

	tools := cfg.ResolveToolNames(executor.ToolCapabilities)
	if len(tools) == 0 {
		t.Fatal("executor should resolve to non-empty tool list")
	}

	// Expected builtin tools from FileEditing + ShellExecution + FileSearching.
	expectedBuiltin := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"}
	for _, expected := range expectedBuiltin {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("executor tools missing %q, got: %v", expected, tools)
		}
	}

	// Capability names should NOT appear as tools.
	capNames := []string{"FileEditing", "ShellExecution", "FileSearching"}
	for _, cap := range capNames {
		for _, tool := range tools {
			if tool == cap {
				t.Errorf("resolved tools should not contain capability name %q", cap)
			}
		}
	}

	// Auditor should NOT resolve to File/Shell tools (no FileEditing/ShellExecution).
	auditor := cfg.RoleMap()["auditor"]
	auditorTools := cfg.ResolveToolNames(auditor.ToolCapabilities)
	shellTools := []string{"Bash", "Glob", "Grep"}
	for _, shell := range shellTools {
		for _, tool := range auditorTools {
			if tool == shell {
				t.Errorf("auditor should not have %q tool, got: %v", shell, auditorTools)
			}
		}
	}

	// ToolClearance integration: verify the clearance filter restricts correctly.
	registry := builtin.NewRegistry()
	clearance := staff.NewToolClearance(cfg, registry, "executor")
	executorVisible := clearance.Names()
	sort.Strings(executorVisible)
	for _, expected := range expectedBuiltin {
		found := false
		for _, name := range executorVisible {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ToolClearance(executor) missing %q, visible: %v", expected, executorVisible)
		}
	}

	// Switch to auditor — Bash should disappear.
	clearance.SetRole("auditor")
	auditorVisible := clearance.Names()
	for _, name := range auditorVisible {
		if name == "Bash" {
			t.Fatal("ToolClearance(auditor) should not expose Bash")
		}
		if name == "Glob" {
			t.Fatal("ToolClearance(auditor) should not expose Glob")
		}
	}
}
