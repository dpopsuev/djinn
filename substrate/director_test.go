package substrate

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/uniform"

	troupe "github.com/dpopsuev/troupe"
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

func TestLocalDirector_ImplementsDirector(t *testing.T) {
	d := NewLocalDirector(DefaultScheduler())
	ch, err := d.Direct(context.Background(), nil)
	if err != nil {
		t.Fatalf("Direct: %v", err)
	}

	var events []troupe.Event
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	if events[0].Kind != troupe.Started {
		t.Fatalf("first event kind = %q, want started", events[0].Kind)
	}
}

func TestLocalDirector_SchedulerAccessor(t *testing.T) {
	s := DefaultScheduler()
	d := NewLocalDirector(s)

	if d.Scheduler() != s {
		t.Fatal("Scheduler() should return the underlying scheduler")
	}

	got := d.Scheduler().NextRole(uniform.SignalGatePassed, "executor")
	if got != "inspector" {
		t.Fatalf("via accessor: got %q, want inspector", got)
	}
}

// stubScheduler verifies the interface is swappable.
type stubScheduler struct {
	called bool
	result string
}

func (s *stubScheduler) NextRole(_ uniform.Signal, _ string) string {
	s.called = true
	return s.result
}

func TestLocalDirector_CustomScheduler(t *testing.T) {
	stub := &stubScheduler{result: "custom-role"}
	d := NewLocalDirector(stub)

	got := d.Scheduler().NextRole(uniform.SignalTasksPlanned, "gensec")
	if got != "custom-role" {
		t.Fatalf("custom scheduler: got %q, want custom-role", got)
	}
	if !stub.called {
		t.Fatal("custom scheduler was not called")
	}
}
