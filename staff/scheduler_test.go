package staff

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
	// Gate fail and inspector reject both loop back to executor
	if NextRole(SignalGateFailed) != "executor" {
		t.Fatal("gate fail should return to executor")
	}
	if NextRole(SignalInspectorRejected) != "executor" {
		t.Fatal("inspector reject should return to executor")
	}
}

func TestNextRole_ExecutorDoneIsNotARole(t *testing.T) {
	// Executor done triggers the gate, not a role switch
	if NextRole(SignalExecutorDone) != "" {
		t.Fatal("executor done should return empty (gate fires, not role)")
	}
}
