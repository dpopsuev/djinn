package orchestrator

import (
	"testing"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/tier"
)

func TestEventKind_String(t *testing.T) {
	tests := []struct {
		kind EventKind
		want string
	}{
		{StageStarted, "stage_started"},
		{StageCompleted, "stage_completed"},
		{StageFailed, "stage_failed"},
		{GatePassed, "gate_passed"},
		{GateFailed, "gate_failed"},
		{ExecutionDone, "execution_done"},
		{EventKind(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Fatalf("EventKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestEvent_Construction(t *testing.T) {
	e := Event{
		ExecID:  "exec-1",
		Kind:    StageStarted,
		Stage:   "lint",
		Message: "starting lint",
	}
	if e.ExecID != "exec-1" {
		t.Fatalf("ExecID = %q, want %q", e.ExecID, "exec-1")
	}
	if e.Kind != StageStarted {
		t.Fatalf("Kind = %v, want StageStarted", e.Kind)
	}
}

func TestWorkPlan_Construction(t *testing.T) {
	plan := WorkPlan{
		ID: "plan-1",
		Stages: []Stage{
			{
				Name:   "code",
				Scope:  tier.Scope{Level: tier.Mod, Name: "auth"},
				Driver: driver.DriverConfig{Model: "claude-opus-4-6"},
				Gate:   gate.GateConfig{Name: "lint", Severity: "blocking"},
				Prompt: "implement auth",
			},
		},
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("len(Stages) = %d, want 1", len(plan.Stages))
	}
	if plan.Stages[0].Name != "code" {
		t.Fatalf("Stage.Name = %q, want %q", plan.Stages[0].Name, "code")
	}
}

func TestExternalInput_Construction(t *testing.T) {
	input := ExternalInput{
		ExecID:  "exec-1",
		Stage:   "review",
		Payload: map[string]string{"action": "approve"},
	}
	if input.ExecID != "exec-1" {
		t.Fatalf("ExecID = %q, want %q", input.ExecID, "exec-1")
	}
}
