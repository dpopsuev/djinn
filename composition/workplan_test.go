package composition

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/tier"
)

func TestToWorkPlan_Solo(t *testing.T) {
	f, err := Instantiate(TemplateSolo(), "pkg/auth", Budget{Tokens: 50000, WallClock: 5 * time.Minute})
	if err != nil {
		t.Fatalf("Instantiate: %v", err)
	}

	plan := ToWorkPlan(f, "plan-1")
	if plan.ID != "plan-1" {
		t.Fatalf("ID = %q, want %q", plan.ID, "plan-1")
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("Stages = %d, want 1", len(plan.Stages))
	}
	if plan.Stages[0].Scope.Level != tier.Mod {
		t.Fatalf("executor scope level = %v, want Mod", plan.Stages[0].Scope.Level)
	}
	if plan.Stages[0].TokenBudget != 50000 {
		t.Fatalf("TokenBudget = %d, want 50000", plan.Stages[0].TokenBudget)
	}
}

func TestToWorkPlan_Duo(t *testing.T) {
	f, err := Instantiate(TemplateDuo(), "pkg/auth", Budget{Tokens: 100000})
	if err != nil {
		t.Fatalf("Instantiate: %v", err)
	}

	plan := ToWorkPlan(f, "plan-2")
	// Reviewer + Executor = 2 stages (observers excluded)
	if len(plan.Stages) != 2 {
		t.Fatalf("Stages = %d, want 2", len(plan.Stages))
	}
	// Reviewer gets Com tier
	if plan.Stages[0].Scope.Level != tier.Com {
		t.Fatalf("reviewer scope = %v, want Com", plan.Stages[0].Scope.Level)
	}
	// Executor gets Mod tier
	if plan.Stages[1].Scope.Level != tier.Mod {
		t.Fatalf("executor scope = %v, want Mod", plan.Stages[1].Scope.Level)
	}
}

func TestToWorkPlan_ObserversExcluded(t *testing.T) {
	f := Formation{
		Name: "with-observer",
		Units: []Unit{
			{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}, Budget: Budget{Tokens: 50000}},
			{Role: RoleObserver, Scope: UnitScope{RO: []string{"pkg/auth"}}},
		},
	}

	plan := ToWorkPlan(f, "plan-3")
	if len(plan.Stages) != 1 {
		t.Fatalf("Stages = %d, want 1 (observer excluded)", len(plan.Stages))
	}
}
