package uniform

import (
	"context"
	"testing"

	troupe "github.com/dpopsuev/troupe"
)

func TestTransitionScheduler_DefaultPipeline(t *testing.T) {
	s := DefaultScheduler()

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
		got := s.NextRole(tt.signal, "")
		if got != tt.want {
			t.Errorf("NextRole(%d) = %q, want %q", tt.signal, got, tt.want)
		}
	}
}

func TestTransitionScheduler_CustomRole(t *testing.T) {
	custom := []Transition{
		{Signal: SignalTasksPlanned, ToRole: "cogs"},
		{Signal: SignalPromptReceived, ToRole: "gensec"},
	}
	s := NewTransitionScheduler(custom)

	if got := s.NextRole(SignalTasksPlanned, ""); got != "cogs" {
		t.Fatalf("custom transition: got %q, want cogs", got)
	}
}

func TestTransitionScheduler_FromRoleSpecific(t *testing.T) {
	transitions := []Transition{
		{Signal: SignalGatePassed, FromRole: "executor", ToRole: "inspector"},
		{Signal: SignalGatePassed, FromRole: "cogs", ToRole: "gensec"},
		{Signal: SignalGatePassed, ToRole: "auditor"}, // fallback for other roles
	}
	s := NewTransitionScheduler(transitions)

	if got := s.NextRole(SignalGatePassed, "executor"); got != "inspector" {
		t.Fatalf("from executor: got %q, want inspector", got)
	}
	if got := s.NextRole(SignalGatePassed, "cogs"); got != "gensec" {
		t.Fatalf("from cogs: got %q, want gensec", got)
	}
	if got := s.NextRole(SignalGatePassed, "unknown"); got != "auditor" {
		t.Fatalf("from unknown: got %q, want auditor (fallback)", got)
	}
}

func TestTransitionScheduler_UnknownSignalFallback(t *testing.T) {
	s := DefaultScheduler()
	if got := s.NextRole(Signal(999), ""); got != "gensec" {
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

	got := d.Scheduler().NextRole(SignalGatePassed, "executor")
	if got != "inspector" {
		t.Fatalf("via accessor: got %q, want inspector", got)
	}
}

// stubScheduler verifies the interface is swappable.
type stubScheduler struct {
	called bool
	result string
}

func (s *stubScheduler) NextRole(_ Signal, _ string) string {
	s.called = true
	return s.result
}

func TestLocalDirector_CustomScheduler(t *testing.T) {
	stub := &stubScheduler{result: "custom-role"}
	d := NewLocalDirector(stub)

	got := d.Scheduler().NextRole(SignalTasksPlanned, "gensec")
	if got != "custom-role" {
		t.Fatalf("custom scheduler: got %q, want custom-role", got)
	}
	if !stub.called {
		t.Fatal("custom scheduler was not called")
	}
}
