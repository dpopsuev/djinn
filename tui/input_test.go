package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Isolated InputPanel tests: zero imports from cli/repl, agent, driver, session ---

func TestInputPanel_SetValueViaMessage(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputSetValueMsg{"hello"})
	if p.Value() != "hello" {
		t.Fatalf("value = %q, want hello", p.Value())
	}
}

func TestInputPanel_ResetViaMessage(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputSetValueMsg{"hello"})
	p.Update(InputResetMsg{})
	if p.Value() != "" {
		t.Fatalf("value = %q after reset", p.Value())
	}
}

func TestInputPanel_FocusBlurViaMessage(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputBlurMsg{})
	if p.Focused() {
		t.Fatal("should be blurred")
	}
	p.Update(InputFocusMsg{})
	if !p.Focused() {
		t.Fatal("should be focused")
	}
}

func TestInputPanel_SubmitEmitsMsg(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputSetValueMsg{"hello"})
	p.SetFocus(true)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should produce a command")
	}
	msg := cmd()
	submit, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if submit.Value != "hello" {
		t.Fatalf("submit = %q, want hello", submit.Value)
	}
	if p.Value() != "" {
		t.Fatalf("value should be cleared after submit, got %q", p.Value())
	}
}

func TestInputPanel_SubmitEmpty_NoCmd(t *testing.T) {
	p := NewInputPanel()
	p.SetFocus(true)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty submit should produce no command")
	}
}

func TestInputPanel_AddHistoryViaMessage(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputAddHistoryMsg{"first"})
	p.Update(InputAddHistoryMsg{"second"})
	p.HistoryUp()
	if p.Value() != "second" {
		t.Fatalf("history up = %q, want second", p.Value())
	}
}

func TestInputPanel_CompletionsViaMessage(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputSetCompletionsMsg{[]string{"/help", "/config", "/clear"}})
	p.SetValue("/he")
	handled, cmd := p.TabComplete()
	if !handled {
		t.Fatal("should handle /he prefix")
	}
	// Single match "/help" — auto-executes via SubmitMsg.
	if cmd == nil {
		t.Fatal("single match should auto-execute")
	}
	msg := cmd()
	submit, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if submit.Value != "/help" {
		t.Fatalf("submit = %q, want /help", submit.Value)
	}
}

func TestInputPanel_TabComplete_SingleMatch_AutoExecute(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputSetCompletionsMsg{[]string{"/help", "/config"}})
	p.SetFocus(true)
	p.Update(InputSetValueMsg{"/hel"})
	handled, cmd := p.TabComplete()
	if !handled {
		t.Fatal("should handle /hel prefix")
	}
	if cmd == nil {
		t.Fatal("single match should auto-execute via SubmitMsg")
	}
	msg := cmd()
	submit, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if submit.Value != "/help" {
		t.Fatalf("submit = %q, want /help", submit.Value)
	}
}

func TestInputPanel_TabComplete_MultiMatch_NoAutoExecute(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputSetCompletionsMsg{[]string{"/config", "/config-save", "/clear"}})
	p.SetFocus(true)
	p.Update(InputSetValueMsg{"/config"})
	_, cmd := p.TabComplete()
	if cmd != nil {
		t.Fatal("multiple matches should NOT auto-execute")
	}
}

func TestInputPanel_ResizeViaMessage(t *testing.T) {
	p := NewInputPanel()
	p.Update(ResizeMsg{Width: 120, Height: 5})
	// Verify no panic — textarea accepted the resize.
	view := p.View(120)
	if view == "" {
		t.Fatal("view should not be empty after resize")
	}
}

func TestInputPanel_PredictionFromHistory(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputAddHistoryMsg{"hello world"})
	p.Update(InputAddHistoryMsg{"help me"})
	p.SetFocus(true)
	p.Update(InputSetValueMsg{"hel"})
	// Trigger prediction update by simulating a keystroke.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	// Prediction should match most recent history entry with prefix "hell"
	// Actually "hell" matches "hello world"
	pred := p.Prediction()
	if pred != "hello world" {
		t.Fatalf("prediction = %q, want 'hello world'", pred)
	}
}

func TestInputPanel_PredictionEmpty(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputAddHistoryMsg{"hello"})
	p.SetFocus(true)
	p.Update(InputSetValueMsg{"xyz"})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if p.Prediction() != "" {
		t.Fatalf("no history match, prediction should be empty, got %q", p.Prediction())
	}
}

func TestInputPanel_AcceptPrediction(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputAddHistoryMsg{"hello world"})
	p.SetFocus(true)
	p.Update(InputSetValueMsg{"hel"})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !p.AcceptPrediction() {
		t.Fatal("should accept prediction")
	}
	if p.Value() != "hello world" {
		t.Fatalf("value = %q, want 'hello world'", p.Value())
	}
}

func TestInputPanel_UnfocusedIgnoresKeys(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputBlurMsg{})
	p.Update(InputSetValueMsg{"hello"})
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("unfocused panel should not emit SubmitMsg")
	}
}
