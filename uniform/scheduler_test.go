package uniform

import "testing"

func TestNextRole_Deterministic(t *testing.T) {
	tests := []struct {
		signal Signal
		want   string
	}{
		{SignalPromptReceived, "gensec"},
		{SignalNeedCaptured, "auditor"},
		{SignalSpecStamped, "scheduler"},
		{SignalTasksPlanned, "executor"},
		{SignalExecutorDone, ""},
		{SignalGatePassed, "inspector"},
		{SignalGateFailed, "executor"},
		{SignalInspectorApproved, "gensec"},
		{SignalInspectorRejected, "executor"},
	}
	for _, tt := range tests {
		got := NextRole(tt.signal)
		if got != tt.want {
			t.Errorf("NextRole(%d) = %q, want %q", tt.signal, got, tt.want)
		}
	}
}

func TestNextRole_UnknownSignalDefaultsToGenSec(t *testing.T) {
	got := NextRole(Signal(999))
	if got != "gensec" {
		t.Fatalf("unknown signal = %q, want gensec", got)
	}
}

func TestNextRole_GateFailLoopsToExecutor(t *testing.T) {
	if NextRole(SignalGateFailed) != "executor" {
		t.Fatal("gate fail should return to executor")
	}
	if NextRole(SignalInspectorRejected) != "executor" {
		t.Fatal("inspector reject should return to executor")
	}
}

func TestNextRole_ExecutorDoneIsNotARole(t *testing.T) {
	if NextRole(SignalExecutorDone) != "" {
		t.Fatal("executor done should return empty (gate fires, not role)")
	}
}

func TestNextRole_CustomTransitions(t *testing.T) {
	custom := []Transition{
		{Signal: SignalTasksPlanned, ToRole: "cogs"},
		{Signal: SignalPromptReceived, ToRole: "gensec"},
		{Signal: SignalGatePassed, ToRole: "inspector"},
	}

	// Custom role routes correctly without code changes.
	if got := NextRole(SignalTasksPlanned, custom); got != "cogs" {
		t.Fatalf("custom transition: got %q, want cogs", got)
	}

	// Other signals still work.
	if got := NextRole(SignalPromptReceived, custom); got != "gensec" {
		t.Fatalf("standard transition: got %q, want gensec", got)
	}

	// Unknown signal falls back to gensec.
	if got := NextRole(Signal(999), custom); got != "gensec" {
		t.Fatalf("unknown signal: got %q, want gensec", got)
	}
}

func TestNextRole_DefaultTransitionsBackwardCompat(t *testing.T) {
	// Calling without transitions uses defaults — backward compatible.
	defaults := DefaultTransitions()
	for _, tr := range defaults {
		withDefault := NextRole(tr.Signal)
		withExplicit := NextRole(tr.Signal, defaults)
		if withDefault != withExplicit {
			t.Errorf("signal %d: default=%q explicit=%q", tr.Signal, withDefault, withExplicit)
		}
	}
}
