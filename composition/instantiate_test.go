package composition

import (
	"testing"
	"time"
)

func TestInstantiate_Solo(t *testing.T) {
	tmpl := TemplateSolo()
	f, err := Instantiate(tmpl, "pkg/auth", Budget{Tokens: 50000, WallClock: 10 * time.Minute})
	if err != nil {
		t.Fatalf("Instantiate: %v", err)
	}

	if f.Name != "solo" {
		t.Fatalf("Name = %q, want %q", f.Name, "solo")
	}
	if len(f.Units) != 1 {
		t.Fatalf("Units = %d, want 1", len(f.Units))
	}

	u := f.Units[0]
	if u.Scope.RW[0] != "pkg/auth" {
		t.Fatalf("RW[0] = %q, want %q (scope substituted)", u.Scope.RW[0], "pkg/auth")
	}
	if u.Budget.Tokens != 50000 {
		t.Fatalf("Tokens = %d, want 50000", u.Budget.Tokens)
	}
	if u.TerminatesWhen.Target != "pkg/auth" {
		t.Fatalf("Target = %q, want %q", u.TerminatesWhen.Target, "pkg/auth")
	}
}

func TestInstantiate_Duo_BudgetDivision(t *testing.T) {
	tmpl := TemplateDuo()
	f, err := Instantiate(tmpl, "pkg/auth", Budget{Tokens: 100000, WallClock: time.Hour})
	if err != nil {
		t.Fatalf("Instantiate: %v", err)
	}

	// 2 non-observer units, budget split equally
	if f.Units[0].Budget.Tokens != 50000 {
		t.Fatalf("reviewer tokens = %d, want 50000", f.Units[0].Budget.Tokens)
	}
	if f.Units[1].Budget.Tokens != 50000 {
		t.Fatalf("executor tokens = %d, want 50000", f.Units[1].Budget.Tokens)
	}
}

func TestInstantiate_Validates(t *testing.T) {
	// Bad template: overlapping RW scopes after substitution
	tmpl := Formation{
		Name: "bad",
		Units: []Unit{
			{Role: RoleExecutor, Scope: UnitScope{RW: []string{templateScopeVar}}},
			{Role: RoleExecutor, Scope: UnitScope{RW: []string{templateScopeVar}}},
		},
	}
	_, err := Instantiate(tmpl, "pkg/auth", Budget{Tokens: 100000})
	if err == nil {
		t.Fatal("should fail validation (overlapping scopes)")
	}
}
