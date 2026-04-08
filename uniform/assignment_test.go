package uniform

import (
	"slices"
	"testing"
)

func TestPrimordialGenSec(t *testing.T) {
	a := PrimordialGenSec()

	if a.Role != RoleGenSec {
		t.Fatalf("Role = %q, want %q", a.Role, RoleGenSec)
	}
	if a.Mode != ModeAuto {
		t.Fatalf("Mode = %q, want %q", a.Mode, ModeAuto)
	}
	if a.Persona != RolePersona[RoleGenSec] {
		t.Fatalf("Persona = %q, want %q", a.Persona, RolePersona[RoleGenSec])
	}

	// Must have capabilities from DefaultConfig gensec role.
	cfg := DefaultConfig()
	roles := cfg.RoleMap()
	wantCaps := roles[RoleGenSec].ToolCapabilities
	if len(a.Capabilities) != len(wantCaps) {
		t.Fatalf("Capabilities len = %d, want %d", len(a.Capabilities), len(wantCaps))
	}
	for i, c := range a.Capabilities {
		if c != wantCaps[i] {
			t.Errorf("Capabilities[%d] = %q, want %q", i, c, wantCaps[i])
		}
	}

	// Global scope: read and write everywhere.
	if len(a.Scope.ReadPaths) != 1 || a.Scope.ReadPaths[0] != "/" {
		t.Fatalf("ReadPaths = %v, want [/]", a.Scope.ReadPaths)
	}
	if len(a.Scope.WritePaths) != 1 || a.Scope.WritePaths[0] != "/" {
		t.Fatalf("WritePaths = %v, want [/]", a.Scope.WritePaths)
	}
}

func TestBroker(t *testing.T) {
	b := Broker()

	if b.Role != RoleGenSec {
		t.Fatalf("Role = %q, want %q", b.Role, RoleGenSec)
	}
	if b.Mode != ModeAgent {
		t.Fatalf("Mode = %q, want %q", b.Mode, ModeAgent)
	}
	if b.Persona != RolePersona[RoleGenSec] {
		t.Fatalf("Persona = %q, want %q", b.Persona, RolePersona[RoleGenSec])
	}

	// Broker has intake-only capabilities — no execution tools.
	executionTools := map[string]bool{
		"FileEditing":    true,
		"ShellExecution": true,
		"FileSearching":  true,
		"QualityGating":  true,
	}
	for _, c := range b.Capabilities {
		if executionTools[c] {
			t.Errorf("Broker should not have execution capability %q", c)
		}
	}

	// Broker can read everywhere but write nowhere.
	if len(b.Scope.ReadPaths) != 1 || b.Scope.ReadPaths[0] != "/" {
		t.Fatalf("ReadPaths = %v, want [/]", b.Scope.ReadPaths)
	}
	if b.Scope.WritePaths != nil {
		t.Fatalf("WritePaths = %v, want nil", b.Scope.WritePaths)
	}
}

func TestSecretary(t *testing.T) {
	scope := AssignmentScope{
		ReadPaths:  []string{"/src/api", "/src/lib"},
		WritePaths: []string{"/src/api"},
	}
	s := Secretary(scope)

	if s.Role != RoleExecutor {
		t.Fatalf("Role = %q, want %q", s.Role, RoleExecutor)
	}
	if s.Mode != ModeAgent {
		t.Fatalf("Mode = %q, want %q", s.Mode, ModeAgent)
	}
	if s.Persona != RolePersona[RoleExecutor] {
		t.Fatalf("Persona = %q, want %q", s.Persona, RolePersona[RoleExecutor])
	}

	// Scope passed through.
	if len(s.Scope.ReadPaths) != 2 {
		t.Fatalf("ReadPaths = %v, want 2 paths", s.Scope.ReadPaths)
	}
	if s.Scope.ReadPaths[0] != "/src/api" || s.Scope.ReadPaths[1] != "/src/lib" {
		t.Fatalf("ReadPaths = %v, want [/src/api /src/lib]", s.Scope.ReadPaths)
	}
	if len(s.Scope.WritePaths) != 1 || s.Scope.WritePaths[0] != "/src/api" {
		t.Fatalf("WritePaths = %v, want [/src/api]", s.Scope.WritePaths)
	}

	// Secretary has execution capabilities.
	for _, required := range []string{"FileEditing", "ShellExecution", "FileSearching"} {
		if !slices.Contains(s.Capabilities, required) {
			t.Errorf("Secretary missing required capability %q", required)
		}
	}
}

func TestFirstSplit(t *testing.T) {
	primordial := PrimordialGenSec()
	broker := Broker()
	sec1 := Secretary(AssignmentScope{
		ReadPaths:  []string{"/src/api"},
		WritePaths: []string{"/src/api"},
	})
	sec2 := Secretary(AssignmentScope{
		ReadPaths:  []string{"/src/lib"},
		WritePaths: []string{"/src/lib"},
	})

	// Combined capabilities of broker + secretaries must cover primordial.
	splitCaps := make(map[string]bool)
	for _, c := range broker.Capabilities {
		splitCaps[c] = true
	}
	for _, c := range sec1.Capabilities {
		splitCaps[c] = true
	}
	for _, c := range sec2.Capabilities {
		splitCaps[c] = true
	}
	for _, c := range primordial.Capabilities {
		if !splitCaps[c] {
			t.Errorf("primordial capability %q not covered by split", c)
		}
	}

	// Secretary scopes must be disjoint.
	for _, rw1 := range sec1.Scope.WritePaths {
		for _, rw2 := range sec2.Scope.WritePaths {
			if rw1 == rw2 {
				t.Errorf("secretary scopes overlap: %q", rw1)
			}
		}
	}

	// Broker writes nowhere; secretaries each write to their own scope.
	if broker.Scope.WritePaths != nil {
		t.Errorf("broker should have nil WritePaths, got %v", broker.Scope.WritePaths)
	}
	if len(sec1.Scope.WritePaths) == 0 {
		t.Error("secretary 1 should have WritePaths")
	}
	if len(sec2.Scope.WritePaths) == 0 {
		t.Error("secretary 2 should have WritePaths")
	}
}

func TestDefaultAssignmentScheduler(t *testing.T) {
	sched := DefaultAssignmentScheduler()

	tests := []struct {
		gear      Gear
		wantRoles []string
	}{
		{GearE1, []string{RoleInspector}},
		{GearE2, []string{RoleScheduler, RoleInspector}},
		{GearE3, []string{RoleAuditor, RoleScheduler, RoleInspector}},
	}

	cfg := DefaultConfig()
	roles := cfg.RoleMap()

	for _, tt := range tests {
		assignments := sched.PlanAssignments(tt.gear)
		if len(assignments) != len(tt.wantRoles) {
			t.Errorf("PlanAssignments(%q) returned %d, want %d",
				tt.gear, len(assignments), len(tt.wantRoles))
			continue
		}

		for i, a := range assignments {
			wantRole := tt.wantRoles[i]
			if a.Role != wantRole {
				t.Errorf("PlanAssignments(%q)[%d].Role = %q, want %q",
					tt.gear, i, a.Role, wantRole)
			}

			// Verify enrichment: mode and capabilities match DefaultConfig.
			role := roles[wantRole]
			if a.Mode != role.Mode {
				t.Errorf("PlanAssignments(%q)[%d].Mode = %q, want %q",
					tt.gear, i, a.Mode, role.Mode)
			}
			if len(a.Capabilities) != len(role.ToolCapabilities) {
				t.Errorf("PlanAssignments(%q)[%d].Capabilities len = %d, want %d",
					tt.gear, i, len(a.Capabilities), len(role.ToolCapabilities))
			}

			// Verify persona is set.
			wantPersona := RolePersona[wantRole]
			if a.Persona != wantPersona {
				t.Errorf("PlanAssignments(%q)[%d].Persona = %q, want %q",
					tt.gear, i, a.Persona, wantPersona)
			}
		}
	}
}

func TestAssignmentSchedulerNone(t *testing.T) {
	sched := DefaultAssignmentScheduler()

	noAssignmentGears := []Gear{GearNone, GearRead, GearPlan, GearE0, GearAuto}
	for _, g := range noAssignmentGears {
		assignments := sched.PlanAssignments(g)
		if assignments != nil {
			t.Errorf("PlanAssignments(%q) = %v, want nil", g, assignments)
		}
	}
}
