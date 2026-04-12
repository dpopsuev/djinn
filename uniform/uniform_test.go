package uniform

import (
	"strings"
	"testing"
)

func TestNewUniform_GenSec(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()
	allTools := []string{
		"Read", "Write", "Edit", "Bash", "Glob", "Grep",
		"git", "assignment", "discourse", "plan", "observe", "latency",
	}

	u := NewUniform("gensec", []string{"director", "manager"}, reg, reqs, allTools, "plan", "opus", "You are GenSec.")

	if u.Persona() != "gensec" {
		t.Errorf("persona = %q, want gensec", u.Persona())
	}

	// GenSec (director+manager) has: shell, observe, coordinate, communicate, work
	if !u.HasCapability(CapShell) {
		t.Error("gensec should have shell")
	}
	if !u.HasCapability(CapCoordinate) {
		t.Error("gensec should have coordinate")
	}
	if !u.HasCapability(CapObserve) {
		t.Error("gensec should have observe")
	}

	// GenSec should NOT have read, write, code, vcs
	if u.HasCapability(CapRead) {
		t.Error("gensec should NOT have read")
	}
	if u.HasCapability(CapWrite) {
		t.Error("gensec should NOT have write")
	}

	// Tools: Bash (shell), assignment (coordinate), discourse (communicate),
	// plan (work), observe + latency (observe). NOT: Read, Write, Edit, Glob, Grep, git
	if !u.HasTool("Bash") {
		t.Error("gensec should have Bash")
	}
	if !u.HasTool("assignment") {
		t.Error("gensec should have assignment")
	}
	if !u.HasTool("discourse") {
		t.Error("gensec should have discourse")
	}
	if u.HasTool("Read") {
		t.Error("gensec should NOT have Read")
	}
	if u.HasTool("Write") {
		t.Error("gensec should NOT have Write")
	}
	if u.HasTool("git") {
		t.Error("gensec should NOT have git")
	}
}

func TestNewUniform_Coder(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()
	allTools := []string{
		"Read", "Write", "Edit", "Bash", "Glob", "Grep",
		"git", "assignment", "discourse", "plan",
	}

	u := NewUniform("coder-1", []string{"developer"}, reg, reqs, allTools, "agent", "sonnet", "You are a Coder.")

	// Coder has: Read, Write, Edit, Glob, Grep, git, discourse, plan
	// Coder does NOT have: Bash, assignment
	if u.HasTool("Bash") {
		t.Error("coder should NOT have Bash")
	}
	if u.HasTool("assignment") {
		t.Error("coder should NOT have assignment")
	}
	if !u.HasTool("Read") {
		t.Error("coder should have Read")
	}
	if !u.HasTool("git") {
		t.Error("coder should have git")
	}
	if !u.HasTool("discourse") {
		t.Error("coder should have discourse (communicate is in agent base)")
	}

	if u.Mode() != "agent" {
		t.Errorf("mode = %q, want agent", u.Mode())
	}
	if u.Model() != "sonnet" {
		t.Errorf("model = %q, want sonnet", u.Model())
	}
}

func TestNewUniform_Operator(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()
	allTools := []string{
		"Read", "Write", "Edit", "Bash", "Glob", "Grep",
		"git", "assignment", "discourse", "plan", "observe",
	}

	u := NewUniform("operator", []string{"operator"}, reg, reqs, allTools, "auto", "opus", "You are the Operator.")

	// Operator has all capabilities — should see ALL tools
	for _, tool := range allTools {
		if !u.HasTool(tool) {
			t.Errorf("operator should have %s", tool)
		}
	}
}

func TestNewUniform_DeniedTools(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()
	allTools := []string{"Read", "Write", "Bash", "assignment"}

	u := NewUniform("coder-1", []string{"developer"}, reg, reqs, allTools, "agent", "", "")

	// Coder denied Bash (shell) and assignment (coordinate)
	denied := u.Denied()
	if len(denied) != 2 {
		t.Errorf("expected 2 denied tools, got %d: %v", len(denied), denied)
	}
}

func TestNewUniform_WarningOnUnknownRole(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()

	u := NewUniform("test", []string{"nonexistent"}, reg, reqs, []string{"Read"}, "", "", "")

	if len(u.Warnings()) == 0 {
		t.Error("expected warning for unknown role")
	}
	found := false
	for _, w := range u.Warnings() {
		if strings.Contains(w, "unknown role") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'unknown role' warning, got: %v", u.Warnings())
	}
}

func TestNewUniform_NoWarningsOnValidConfig(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()
	allTools := []string{"Read", "Write", "discourse", "plan"}

	u := NewUniform("coder", []string{"developer"}, reg, reqs, allTools, "agent", "", "")

	if len(u.Warnings()) != 0 {
		t.Errorf("expected no warnings, got: %v", u.Warnings())
	}
}

func TestUniform_SystemContext_ListsOnlyAllowedTools(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	reqs := DefaultToolRequirements()
	allTools := []string{"Read", "Write", "Bash", "assignment", "discourse", "plan"}

	u := NewUniform("coder-1", []string{"developer"}, reg, reqs, allTools, "agent", "sonnet", "You are a Coder.")
	ctx := u.SystemContext()

	// Should mention allowed tools
	if !strings.Contains(ctx, "Read") {
		t.Error("system context should mention Read")
	}
	if !strings.Contains(ctx, "discourse") {
		t.Error("system context should mention discourse")
	}

	// Should NOT mention tools the coder can't use
	if strings.Contains(ctx, "Bash") {
		t.Error("system context should NOT mention Bash for coder")
	}
	if strings.Contains(ctx, "assignment") {
		t.Error("system context should NOT mention assignment for coder")
	}

	// Should include the prompt
	if !strings.Contains(ctx, "You are a Coder") {
		t.Error("system context should include persona prompt")
	}

	// Should include the restriction
	if !strings.Contains(ctx, "Do not attempt") {
		t.Error("system context should tell agent not to use unlisted tools")
	}
}
