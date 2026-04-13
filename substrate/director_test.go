package substrate

import (
	"testing"

	"github.com/dpopsuev/djinn/uniform"
)

func TestTransitionScheduler_DefaultPipeline(t *testing.T) {
	s := DefaultScheduler()

	tests := []struct {
		signal uniform.Signal
		want   string
	}{
		{uniform.SignalPromptReceived, "gensec"},
		{uniform.SignalNeedCaptured, "auditor"},
		{uniform.SignalSpecStamped, "scheduler"},
		{uniform.SignalTasksPlanned, "executor"},
		{uniform.SignalExecutorDone, ""},
		{uniform.SignalGatePassed, "inspector"},
		{uniform.SignalGateFailed, "executor"},
		{uniform.SignalInspectorApproved, "gensec"},
		{uniform.SignalInspectorRejected, "executor"},
	}
	for _, tt := range tests {
		got := s.NextRole(tt.signal, "")
		if got != tt.want {
			t.Errorf("NextRole(%d) = %q, want %q", tt.signal, got, tt.want)
		}
	}
}

func TestTransitionScheduler_CustomRole(t *testing.T) {
	custom := []uniform.Transition{
		{Signal: uniform.SignalTasksPlanned, ToRole: "cogs"},
		{Signal: uniform.SignalPromptReceived, ToRole: "gensec"},
	}
	s := NewTransitionScheduler(custom)

	if got := s.NextRole(uniform.SignalTasksPlanned, ""); got != "cogs" {
		t.Fatalf("custom transition: got %q, want cogs", got)
	}
}

func TestTransitionScheduler_FromRoleSpecific(t *testing.T) {
	transitions := []uniform.Transition{
		{Signal: uniform.SignalGatePassed, FromRole: "executor", ToRole: "inspector"},
		{Signal: uniform.SignalGatePassed, FromRole: "cogs", ToRole: "gensec"},
		{Signal: uniform.SignalGatePassed, ToRole: "auditor"}, // fallback for other roles
	}
	s := NewTransitionScheduler(transitions)

	if got := s.NextRole(uniform.SignalGatePassed, "executor"); got != "inspector" {
		t.Fatalf("from executor: got %q, want inspector", got)
	}
	if got := s.NextRole(uniform.SignalGatePassed, "cogs"); got != "gensec" {
		t.Fatalf("from cogs: got %q, want gensec", got)
	}
	if got := s.NextRole(uniform.SignalGatePassed, "unknown"); got != "auditor" {
		t.Fatalf("from unknown: got %q, want auditor (fallback)", got)
	}
}

func TestTransitionScheduler_UnknownSignalFallback(t *testing.T) {
	s := DefaultScheduler()
	if got := s.NextRole(uniform.Signal(999), ""); got != "gensec" {
		t.Fatalf("unknown signal: got %q, want gensec", got)
	}
}

// stubScheduler verifies the Scheduler interface is swappable.
type stubScheduler struct {
	called bool
	result string
}

func (s *stubScheduler) NextRole(_ uniform.Signal, _ string) string {
	s.called = true
	return s.result
}

func TestTransitionScheduler_CustomSchedulerInterface(t *testing.T) {
	stub := &stubScheduler{result: "custom-role"}

	got := stub.NextRole(uniform.SignalTasksPlanned, "gensec")
	if got != "custom-role" {
		t.Fatalf("custom scheduler: got %q, want custom-role", got)
	}
	if !stub.called {
		t.Fatal("custom scheduler was not called")
	}
}
