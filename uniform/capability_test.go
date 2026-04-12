package uniform

import (
	"sort"
	"testing"
)

func TestResolve_AgentBase(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	caps := reg.Resolve("agent")

	if !HasCapability(caps, CapCommunicate) {
		t.Error("agent should have communicate")
	}
	if !HasCapability(caps, CapWork) {
		t.Error("agent should have work")
	}
	if len(caps) != 2 {
		t.Errorf("agent should have 2 capabilities, got %d", len(caps))
	}
}

func TestResolve_DeveloperComposesAgent(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	caps := reg.Resolve("developer")

	expected := []Capability{CapRead, CapWrite, CapCode, CapVCS, CapCommunicate, CapWork}
	for _, e := range expected {
		if !HasCapability(caps, e) {
			t.Errorf("developer should have %s", e)
		}
	}
	if HasCapability(caps, CapShell) {
		t.Error("developer should NOT have shell")
	}
	if HasCapability(caps, CapCoordinate) {
		t.Error("developer should NOT have coordinate")
	}
}

func TestResolve_DirectorComposesManager(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	caps := reg.Resolve("director")

	// Composed: director → manager → agent. Gets shell + observe + coordinate + communicate + work.
	expected := []Capability{CapShell, CapObserve, CapCoordinate, CapCommunicate, CapWork}
	for _, e := range expected {
		if !HasCapability(caps, e) {
			t.Errorf("director should have %s", e)
		}
	}
}

func TestResolve_OperatorHasAll(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	caps := reg.Resolve("operator")

	// operator = developer + architect + qa + operations + shell
	// Should have ALL 9 capabilities.
	all := BuiltinCapabilities()
	for _, c := range all {
		if !HasCapability(caps, c) {
			t.Errorf("operator should have %s", c)
		}
	}
}

func TestResolve_UnknownRole(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	caps := reg.Resolve("nonexistent")

	if len(caps) != 0 {
		t.Errorf("unknown role should have 0 capabilities, got %d", len(caps))
	}
}

func TestResolve_CycleProtection(t *testing.T) {
	reg := NewRoleRegistry([]RoleDef{
		{Name: "a", Composes: []string{"b"}, Capabilities: []Capability{CapRead}},
		{Name: "b", Composes: []string{"a"}, Capabilities: []Capability{CapWrite}},
	})
	caps := reg.Resolve("a")

	// Should not infinite loop. Should have both capabilities.
	if !HasCapability(caps, CapRead) || !HasCapability(caps, CapWrite) {
		t.Errorf("cycle should resolve with both caps, got %v", caps)
	}
}

func TestResolvePersona_GenSec(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	// GenSec = [director, manager]
	caps := reg.ResolvePersona([]string{"director", "manager"})

	expected := []Capability{CapShell, CapObserve, CapCoordinate, CapCommunicate, CapWork}
	for _, e := range expected {
		if !HasCapability(caps, e) {
			t.Errorf("GenSec should have %s", e)
		}
	}
}

func TestResolvePersona_Coder(t *testing.T) {
	reg := NewRoleRegistry(DefaultRoles())
	// Coder = [developer]
	caps := reg.ResolvePersona([]string{"developer"})

	if HasCapability(caps, CapShell) {
		t.Error("Coder should NOT have shell")
	}
	if HasCapability(caps, CapCoordinate) {
		t.Error("Coder should NOT have coordinate")
	}
	if !HasCapability(caps, CapRead) {
		t.Error("Coder should have read")
	}
}

func TestAllCapabilities(t *testing.T) {
	all := BuiltinCapabilities()
	if len(all) != 9 {
		t.Errorf("expected 9 capabilities, got %d", len(all))
	}

	// Verify sorted uniqueness.
	names := make([]string, len(all))
	for i, c := range all {
		names[i] = string(c)
	}
	sort.Strings(names)
	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			t.Errorf("duplicate capability: %s", names[i])
		}
	}
}
