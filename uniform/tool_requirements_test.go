package uniform

import (
	"sort"
	"testing"
)

func TestToolRequirements_Allowed(t *testing.T) {
	reqs := DefaultToolRequirements()

	// Developer has: read, write, code, vcs, communicate, work
	devCaps := NewRoleRegistry(DefaultRoles()).Resolve("developer")

	// Developer CAN use Read (requires read)
	if !reqs.Allowed("Read", devCaps) {
		t.Error("developer should be allowed Read")
	}

	// Developer CAN use Write (requires write)
	if !reqs.Allowed("Write", devCaps) {
		t.Error("developer should be allowed Write")
	}

	// Developer CAN use git (requires vcs)
	if !reqs.Allowed("git", devCaps) {
		t.Error("developer should be allowed git")
	}

	// Developer CANNOT use Bash (requires shell)
	if reqs.Allowed("Bash", devCaps) {
		t.Error("developer should NOT be allowed Bash")
	}

	// Developer CANNOT use assignment (requires coordinate)
	if reqs.Allowed("assignment", devCaps) {
		t.Error("developer should NOT be allowed assignment")
	}

	// Developer CAN use discourse (requires communicate — agent base has it)
	if !reqs.Allowed("discourse", devCaps) {
		t.Error("developer should be allowed discourse")
	}
}

func TestToolRequirements_Allowed_Director(t *testing.T) {
	reqs := DefaultToolRequirements()

	// Director has: shell, observe, coordinate, communicate, work
	dirCaps := NewRoleRegistry(DefaultRoles()).Resolve("director")

	// Director CAN use Bash (requires shell)
	if !reqs.Allowed("Bash", dirCaps) {
		t.Error("director should be allowed Bash")
	}

	// Director CAN use assignment (requires coordinate)
	if !reqs.Allowed("assignment", dirCaps) {
		t.Error("director should be allowed assignment")
	}

	// Director CANNOT use Write (requires write — director doesn't have write)
	if reqs.Allowed("Write", dirCaps) {
		t.Error("director should NOT be allowed Write")
	}
}

func TestToolRequirements_Allowed_Operator(t *testing.T) {
	reqs := DefaultToolRequirements()

	// Operator has all capabilities
	opCaps := NewRoleRegistry(DefaultRoles()).Resolve("operator")

	// Operator can use everything
	allTools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "git", "assignment", "discourse", "observe"}
	for _, tool := range allTools {
		if !reqs.Allowed(tool, opCaps) {
			t.Errorf("operator should be allowed %s", tool)
		}
	}
}

func TestToolRequirements_Filter(t *testing.T) {
	reqs := DefaultToolRequirements()

	devCaps := NewRoleRegistry(DefaultRoles()).Resolve("developer")
	allTools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "git", "assignment", "discourse", "plan"}

	allowed := reqs.Filter(allTools, devCaps)
	sort.Strings(allowed)

	// Developer should see: Read, Write, Edit, Glob, Grep, git, discourse, plan
	// Developer should NOT see: Bash, assignment
	for _, name := range allowed {
		if name == "Bash" {
			t.Error("filter should exclude Bash for developer")
		}
		if name == "assignment" {
			t.Error("filter should exclude assignment for developer")
		}
	}

	if len(allowed) != 8 {
		t.Errorf("expected 8 allowed tools for developer, got %d: %v", len(allowed), allowed)
	}
}

func TestToolRequirements_UnregisteredToolAllowed(t *testing.T) {
	reqs := DefaultToolRequirements()

	// Tool not in requirements map → unrestricted
	if !reqs.Allowed("unknown_tool", nil) {
		t.Error("unregistered tools should be allowed (unrestricted)")
	}
}

func TestToolRequirements_CustomCapability(t *testing.T) {
	reqs := NewToolRequirements()
	reqs.Set("deploy", Capability("deploy"))

	// Agent without deploy capability
	if reqs.Allowed("deploy", []Capability{CapRead, CapWrite}) {
		t.Error("should not be allowed without deploy capability")
	}

	// Agent with deploy capability
	if !reqs.Allowed("deploy", []Capability{CapRead, Capability("deploy")}) {
		t.Error("should be allowed with deploy capability")
	}
}
