package builders

import (
	"testing"

	"github.com/dpopsuev/djinn/tier"
)

func TestWorkPlanBuilder_Fluent(t *testing.T) {
	plan := NewWorkPlan("plan-1").
		AddStage("code", tier.Scope{Level: tier.Mod, Name: "auth"}, "implement auth").
		AddStage("test", tier.Scope{Level: tier.Mod, Name: "tests"}, "run tests").
		Build()

	if plan.ID != "plan-1" {
		t.Fatalf("ID = %q, want %q", plan.ID, "plan-1")
	}
	if len(plan.Stages) != 2 {
		t.Fatalf("Stages = %d, want 2", len(plan.Stages))
	}
	if plan.Stages[0].Name != "code" {
		t.Fatalf("Stage[0].Name = %q, want %q", plan.Stages[0].Name, "code")
	}
	if plan.Stages[1].Prompt != "run tests" {
		t.Fatalf("Stage[1].Prompt = %q, want %q", plan.Stages[1].Prompt, "run tests")
	}
}

func TestStandardFourTierPlan(t *testing.T) {
	plan := StandardFourTierPlan("std-1")
	if len(plan.Stages) != 4 {
		t.Fatalf("Stages = %d, want 4", len(plan.Stages))
	}

	expected := []string{"analyze", "code", "test", "review"}
	for i, name := range expected {
		if plan.Stages[i].Name != name {
			t.Fatalf("Stage[%d].Name = %q, want %q", i, plan.Stages[i].Name, name)
		}
	}
}
