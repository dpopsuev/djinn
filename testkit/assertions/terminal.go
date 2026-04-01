package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/terminal"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

// AssertStatus checks that the terminal state matches expected values.
func AssertStatus(t *testing.T, d *terminal.Djinn, field string, expected any) {
	t.Helper()
	s := d.Status()
	switch field {
	case "operation":
		if s.Operation != expected.(string) {
			t.Errorf("Operation = %q, want %q", s.Operation, expected)
		}
	case "tokens_in":
		if s.TokensIn != expected.(int) {
			t.Errorf("TokensIn = %d, want %d", s.TokensIn, expected)
		}
	case "turns":
		if s.Turns != expected.(int) {
			t.Errorf("Turns = %d, want %d", s.Turns, expected)
		}
	case "agent_count":
		if s.AgentCount != expected.(int) {
			t.Errorf("AgentCount = %d, want %d", s.AgentCount, expected)
		}
	}
}

// AssertEventReceived checks that the TestOperator received an event of the given kind.
func AssertEventReceived(t *testing.T, op *stubs.TestOperator, kind terminal.ViewEventKind) {
	t.Helper()
	events := op.Events()
	for i := range events {
		if events[i].Kind == kind {
			return
		}
	}
	t.Errorf("expected event %q not found in %d events", kind, len(events))
}
