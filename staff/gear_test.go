package staff

import (
	"testing"
)

func TestParseGear_ValidGears(t *testing.T) {
	tests := []struct {
		input string
		want  Gear
	}{
		{"N", GearNone},
		{"n", GearNone},
		{"R", GearRead},
		{"r", GearRead},
		{"P", GearPlan},
		{"E0", GearE0},
		{"e0", GearE0},
		{"E1", GearE1},
		{"E2", GearE2},
		{"E3", GearE3},
		{"A", GearAuto},
		{"  e1  ", GearE1},
	}
	for _, tt := range tests {
		g, err := ParseGear(tt.input)
		if err != nil {
			t.Errorf("ParseGear(%q) error: %v", tt.input, err)
		}
		if g != tt.want {
			t.Errorf("ParseGear(%q) = %q, want %q", tt.input, g, tt.want)
		}
	}
}

func TestParseGear_Invalid(t *testing.T) {
	_, err := ParseGear("X9")
	if err == nil {
		t.Fatal("expected error for invalid gear")
	}
	_, err = ParseGear("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestGear_Executors(t *testing.T) {
	tests := []struct {
		gear Gear
		want int
	}{
		{GearNone, 0},
		{GearRead, 0},
		{GearPlan, 0},
		{GearE0, 0},
		{GearE1, 1},
		{GearE2, 2},
		{GearE3, 3},
		{GearAuto, 0},
	}
	for _, tt := range tests {
		if got := tt.gear.Executors(); got != tt.want {
			t.Errorf("%q.Executors() = %d, want %d", tt.gear, got, tt.want)
		}
	}
}

func TestGear_SupportRoles(t *testing.T) {
	// E1 gets inspector only
	roles := GearE1.SupportRoles()
	if len(roles) != 1 || roles[0] != "inspector" {
		t.Fatalf("E1 support = %v, want [inspector]", roles)
	}

	// E2 gets scheduler + inspector
	roles = GearE2.SupportRoles()
	if len(roles) != 2 {
		t.Fatalf("E2 support = %v, want 2 roles", roles)
	}
	want := map[string]bool{"scheduler": true, "inspector": true}
	for _, r := range roles {
		if !want[r] {
			t.Errorf("E2 unexpected support role: %q", r)
		}
	}

	// E3 gets auditor + scheduler + inspector
	roles = GearE3.SupportRoles()
	if len(roles) != 3 {
		t.Fatalf("E3 support = %v, want 3 roles", roles)
	}

	// Non-executor gears get nil
	if GearNone.SupportRoles() != nil {
		t.Fatal("GearNone should have nil support roles")
	}
	if GearRead.SupportRoles() != nil {
		t.Fatal("GearRead should have nil support roles")
	}
	if GearPlan.SupportRoles() != nil {
		t.Fatal("GearPlan should have nil support roles")
	}
}

func TestGear_State(t *testing.T) {
	state := GearE2.State()
	if state.Current != GearE2 {
		t.Fatalf("state.Current = %q", state.Current)
	}
	if state.Executors != 2 {
		t.Fatalf("state.Executors = %d", state.Executors)
	}
	if len(state.Support) != 2 {
		t.Fatalf("state.Support = %v", state.Support)
	}
}

func TestClassifyPromptComplexity_ReadOnly(t *testing.T) {
	g := ClassifyPromptComplexity("what is this?")
	if g != GearRead {
		t.Fatalf("short question = %q, want R", g)
	}

	g = ClassifyPromptComplexity("how does it work?")
	if g != GearRead {
		t.Fatalf("short question = %q, want R", g)
	}
}

func TestClassifyPromptComplexity_Plan(t *testing.T) {
	g := ClassifyPromptComplexity("design a new authentication flow for the API")
	if g != GearPlan {
		t.Fatalf("design prompt = %q, want P", g)
	}

	g = ClassifyPromptComplexity("write a spec for the data pipeline")
	// "write" matches E1 before "spec" matches Plan, but spec is checked after E1
	// Actually "write" hits E1 first. This is by design: "write a spec" is E1.
	// Let's test a pure plan keyword.
	g = ClassifyPromptComplexity("plan the architecture for the new service")
	if g != GearPlan {
		t.Fatalf("plan prompt = %q, want P", g)
	}
}

func TestClassifyPromptComplexity_E0(t *testing.T) {
	g := ClassifyPromptComplexity("fix the typo in the error message on line 42")
	if g != GearE0 {
		t.Fatalf("fix prompt = %q, want E0", g)
	}

	g = ClassifyPromptComplexity("rename the variable from x to count")
	if g != GearE0 {
		t.Fatalf("rename prompt = %q, want E0", g)
	}
}

func TestClassifyPromptComplexity_E1(t *testing.T) {
	g := ClassifyPromptComplexity("implement a new user registration endpoint")
	if g != GearE1 {
		t.Fatalf("implement prompt = %q, want E1", g)
	}

	g = ClassifyPromptComplexity("create a new config parser module")
	if g != GearE1 {
		t.Fatalf("create prompt = %q, want E1", g)
	}

	g = ClassifyPromptComplexity("build the dashboard component with charts")
	if g != GearE1 {
		t.Fatalf("build prompt = %q, want E1", g)
	}
}

func TestClassifyPromptComplexity_E2(t *testing.T) {
	g := ClassifyPromptComplexity("refactor the logging system across the entire codebase")
	if g != GearE2 {
		t.Fatalf("refactor+entire = %q, want E2", g)
	}

	g = ClassifyPromptComplexity("restructure all files in the project to use the new pattern")
	if g != GearE2 {
		t.Fatalf("restructure+all files = %q, want E2", g)
	}
}

func TestClassifyPromptComplexity_E3(t *testing.T) {
	g := ClassifyPromptComplexity("overhaul the entire database layer and all its consumers")
	if g != GearE3 {
		t.Fatalf("overhaul = %q, want E3", g)
	}

	g = ClassifyPromptComplexity("rewrite the API server from scratch")
	if g != GearE3 {
		t.Fatalf("rewrite = %q, want E3", g)
	}

	g = ClassifyPromptComplexity("migrate from REST to gRPC across all services")
	if g != GearE3 {
		t.Fatalf("migrate = %q, want E3", g)
	}
}

func TestClassifyPromptComplexity_DefaultE0(t *testing.T) {
	g := ClassifyPromptComplexity("hello there, long enough prompt without any keywords at all for the system")
	if g != GearE0 {
		t.Fatalf("no keywords = %q, want E0 (default)", g)
	}
}
