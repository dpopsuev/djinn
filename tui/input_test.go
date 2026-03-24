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
	if !p.TabComplete() {
		t.Fatal("should handle /he prefix")
	}
	if p.Value() != "/help" {
		t.Fatalf("completed = %q, want /help", p.Value())
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

func TestInputPanel_UnfocusedIgnoresKeys(t *testing.T) {
	p := NewInputPanel()
	p.Update(InputBlurMsg{})
	p.Update(InputSetValueMsg{"hello"})
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("unfocused panel should not emit SubmitMsg")
	}
}
