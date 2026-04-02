// security_test.go — Trust boundary tests: agent output cannot cross into operator input.
// TB-1: Agent output isolation — OutputAppendMsg routes ONLY to OutputPanel.
package widgets

import (
	"testing"

	tui "github.com/dpopsuev/djinn/tui"
)

// TestAgentOutput_GoesToOutputPanel verifies that OutputAppendMsg
// is correctly handled by the OutputPanel and its content appears
// in the rendered view.
func TestAgentOutput_GoesToOutputPanel(t *testing.T) {
	out := NewOutputPanel()
	out.Update(tui.ResizeMsg{Width: 80, Height: 20})

	// Simulate agent output arriving via message bus.
	out.Update(tui.OutputAppendMsg{Line: "agent response line 1"})
	out.Update(tui.OutputAppendMsg{Line: "agent response line 2"})

	if out.LineCount() != 2 {
		t.Fatalf("output panel lines = %d, want 2", out.LineCount())
	}
	if out.Lines()[0] != "agent response line 1" {
		t.Fatalf("line[0] = %q, want 'agent response line 1'", out.Lines()[0])
	}
	if out.Lines()[1] != "agent response line 2" {
		t.Fatalf("line[1] = %q, want 'agent response line 2'", out.Lines()[1])
	}
}

// TestAgentOutput_CannotWriteInputPanel verifies that OutputAppendMsg
// does NOT affect the InputPanel state. This is the trust boundary TB-1:
// an agent cannot inject text into the operator's input area.
func TestAgentOutput_CannotWriteInputPanel(t *testing.T) {
	inp := NewInputPanel()
	inp.SetFocus(true)

	// Record initial state.
	before := inp.Value()

	// Send agent output messages to the input panel — these should be
	// silently ignored because InputPanel.Update does not handle them.
	inp.Update(tui.OutputAppendMsg{Line: "injected by agent"})
	inp.Update(tui.OutputSetLineMsg{Index: 0, Line: "overwrite attempt"})
	inp.Update(tui.OutputAppendLastMsg{Text: "append attempt"})
	inp.Update(tui.OutputSetOverlayMsg{Text: "overlay attempt"})
	inp.Update(tui.OutputClearMsg{})

	after := inp.Value()
	if after != before {
		t.Fatalf("InputPanel value changed from %q to %q after OutputAppendMsg — trust boundary violated", before, after)
	}

	// Also verify that the input panel still functions correctly after
	// receiving alien messages — no silent corruption.
	inp.Update(tui.InputSetValueMsg{Value: "operator typing"})
	if inp.Value() != "operator typing" {
		t.Fatalf("InputPanel broken after alien messages: value = %q", inp.Value())
	}
}
