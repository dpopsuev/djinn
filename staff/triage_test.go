package staff

import (
	"strings"
	"testing"
)

func TestTriagePrompt_SimpleQuestion(t *testing.T) {
	d := TriagePrompt("what is this?", GearAuto, 10.0)

	if d.SuggestedGear != GearRead {
		t.Errorf("simple question gear = %q, want %q", d.SuggestedGear, GearRead)
	}
	if d.State != TriageReady {
		t.Errorf("state = %q, want %q", d.State, TriageReady)
	}
}

func TestTriagePrompt_ImplementTask(t *testing.T) {
	d := TriagePrompt("implement a new user registration endpoint", GearAuto, 10.0)

	if d.SuggestedGear != GearE1 {
		t.Errorf("implement prompt gear = %q, want %q", d.SuggestedGear, GearE1)
	}
	if !strings.Contains(d.Reason, "E1") {
		t.Errorf("reason should mention E1: %q", d.Reason)
	}
}

func TestTriagePrompt_ComplexRefactor(t *testing.T) {
	d := TriagePrompt("refactor the logging system across the entire codebase", GearAuto, 10.0)

	if d.SuggestedGear != GearE2 {
		t.Errorf("refactor prompt gear = %q, want %q", d.SuggestedGear, GearE2)
	}
}

func TestTriagePrompt_MajorOverhaul(t *testing.T) {
	d := TriagePrompt("overhaul the entire database layer", GearAuto, 10.0)

	if d.SuggestedGear != GearE3 {
		t.Errorf("overhaul prompt gear = %q, want %q", d.SuggestedGear, GearE3)
	}
}

func TestTriagePrompt_SmallFix(t *testing.T) {
	d := TriagePrompt("fix the typo in main.go", GearAuto, 10.0)

	if d.SuggestedGear != GearE0 {
		t.Errorf("fix prompt gear = %q, want %q", d.SuggestedGear, GearE0)
	}
}

func TestTriagePrompt_PlanDesign(t *testing.T) {
	d := TriagePrompt("design a new authentication architecture", GearAuto, 10.0)

	if d.SuggestedGear != GearPlan {
		t.Errorf("design prompt gear = %q, want %q", d.SuggestedGear, GearPlan)
	}
}

func TestTriagePrompt_BudgetExhausted(t *testing.T) {
	d := TriagePrompt("overhaul the entire system", GearAuto, 0)

	if d.SuggestedGear != GearRead {
		t.Errorf("zero budget gear = %q, want %q (downshift to R)", d.SuggestedGear, GearRead)
	}
	if !strings.Contains(d.Reason, "budget exhausted") {
		t.Errorf("reason should mention budget exhausted: %q", d.Reason)
	}
}

func TestTriagePrompt_NegativeBudget(t *testing.T) {
	d := TriagePrompt("implement a feature", GearAuto, -1.0)

	if d.SuggestedGear != GearRead {
		t.Errorf("negative budget gear = %q, want %q", d.SuggestedGear, GearRead)
	}
}

func TestTriagePrompt_LowBudget(t *testing.T) {
	d := TriagePrompt("overhaul the entire database layer", GearAuto, 0.30)

	if d.SuggestedGear != GearE0 {
		t.Errorf("low budget gear = %q, want %q (capped at E0)", d.SuggestedGear, GearE0)
	}
	if !strings.Contains(d.Reason, "low budget") {
		t.Errorf("reason should mention low budget: %q", d.Reason)
	}
}

func TestTriagePrompt_LowBudgetSmallTask(t *testing.T) {
	// Small task within budget ceiling — no downshift.
	d := TriagePrompt("what is this?", GearAuto, 0.30)

	if d.SuggestedGear != GearRead {
		t.Errorf("low budget small task gear = %q, want %q", d.SuggestedGear, GearRead)
	}
	// Reason should NOT mention budget constraint since Read < E0 ceiling.
	if strings.Contains(d.Reason, "low budget") {
		t.Errorf("reason should not mention budget for small task: %q", d.Reason)
	}
}

func TestTriagePrompt_SufficientBudget(t *testing.T) {
	d := TriagePrompt("overhaul the database", GearAuto, 5.0)

	if d.SuggestedGear != GearE3 {
		t.Errorf("sufficient budget gear = %q, want %q", d.SuggestedGear, GearE3)
	}
	if strings.Contains(d.Reason, "budget") {
		t.Errorf("reason should not mention budget: %q", d.Reason)
	}
}

func TestTriagePrompt_StateAlwaysReady(t *testing.T) {
	prompts := []string{
		"what is this?",
		"implement a feature",
		"overhaul everything",
	}
	for _, p := range prompts {
		d := TriagePrompt(p, GearAuto, 10.0)
		if d.State != TriageReady {
			t.Errorf("TriagePrompt(%q).State = %q, want %q", p, d.State, TriageReady)
		}
	}
}

func TestDotForState_Idle(t *testing.T) {
	dot := DotForState(TriageIdle)
	if dot.Color != "green" {
		t.Errorf("idle color = %q, want %q", dot.Color, "green")
	}
	if dot.State != TriageIdle {
		t.Errorf("idle state = %q, want %q", dot.State, TriageIdle)
	}
}

func TestDotForState_Analyzing(t *testing.T) {
	dot := DotForState(TriageAnalyzing)
	if dot.Color != "yellow" {
		t.Errorf("analyzing color = %q, want %q", dot.Color, "yellow")
	}
}

func TestDotForState_Routing(t *testing.T) {
	dot := DotForState(TriageRouting)
	if dot.Color != "blue" {
		t.Errorf("routing color = %q, want %q", dot.Color, "blue")
	}
}

func TestDotForState_Ready(t *testing.T) {
	dot := DotForState(TriageReady)
	if dot.Color != "white" {
		t.Errorf("ready color = %q, want %q", dot.Color, "white")
	}
}

func TestDotForState_UnknownState(t *testing.T) {
	dot := DotForState(TriageState("bogus"))
	if dot.Color != "green" {
		t.Errorf("unknown state color = %q, want %q (default green)", dot.Color, "green")
	}
}

func TestDotForState_AllStates(t *testing.T) {
	states := []TriageState{TriageIdle, TriageAnalyzing, TriageRouting, TriageReady}
	expectedColors := []string{"green", "yellow", "blue", "white"}

	for i, s := range states {
		dot := DotForState(s)
		if dot.Color != expectedColors[i] {
			t.Errorf("DotForState(%q).Color = %q, want %q", s, dot.Color, expectedColors[i])
		}
	}
}

func TestGearRank_Ordering(t *testing.T) {
	// Ensure rank ordering is monotonically increasing for the gear progression.
	gears := []Gear{GearNone, GearRead, GearPlan, GearE0, GearE1, GearE2, GearE3}
	for i := 1; i < len(gears); i++ {
		if gearRank(gears[i]) <= gearRank(gears[i-1]) {
			t.Errorf("gearRank(%q)=%d should be > gearRank(%q)=%d",
				gears[i], gearRank(gears[i]), gears[i-1], gearRank(gears[i-1]))
		}
	}
}

func TestConstrainForBudget_NoDownshift(t *testing.T) {
	// E0 is within E1 ceiling.
	got := constrainForBudget(GearE0, GearE1)
	if got != GearE0 {
		t.Errorf("constrainForBudget(E0, E1) = %q, want E0", got)
	}
}

func TestConstrainForBudget_Downshift(t *testing.T) {
	got := constrainForBudget(GearE3, GearE0)
	if got != GearE0 {
		t.Errorf("constrainForBudget(E3, E0) = %q, want E0", got)
	}
}
