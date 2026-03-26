package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDialogPanel_NewCreatesWithTitleAndActions(t *testing.T) {
	d := NewDialogPanel("Confirm", "Are you sure?",
		DialogAction{ID: "allow", Label: "[Allow]"},
		DialogAction{ID: "deny", Label: "[Deny]"},
	)
	if d.ID() != "dialog" {
		t.Fatalf("ID = %q, want %q", d.ID(), "dialog")
	}
	if d.title != "Confirm" {
		t.Fatalf("title = %q, want %q", d.title, "Confirm")
	}
	if len(d.actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(d.actions))
	}
}

func TestDialogPanel_ViewRendersBorderedBox(t *testing.T) {
	d := NewDialogPanel("Confirm", "Are you sure?",
		DialogAction{ID: "allow", Label: "[Allow]"},
	)
	view := d.View(60)
	if !strings.Contains(view, "Confirm") {
		t.Fatal("view should contain title")
	}
	if !strings.Contains(view, "Are you sure?") {
		t.Fatal("view should contain message")
	}
	// Rounded border uses curved corners.
	if !strings.Contains(view, "╭") {
		t.Fatal("view should have rounded border (top-left corner)")
	}
}

func TestDialogPanel_ViewRendersActionButtons(t *testing.T) {
	d := NewDialogPanel("Title", "msg",
		DialogAction{ID: "allow", Label: "[Allow]"},
		DialogAction{ID: "deny", Label: "[Deny]"},
	)
	view := d.View(60)
	if !strings.Contains(view, "[Allow]") {
		t.Fatal("view should contain Allow button")
	}
	if !strings.Contains(view, "[Deny]") {
		t.Fatal("view should contain Deny button")
	}
}

func TestDialogPanel_TabCyclesActions(t *testing.T) {
	d := NewDialogPanel("Title", "msg",
		DialogAction{ID: "allow", Label: "[Allow]"},
		DialogAction{ID: "deny", Label: "[Deny]"},
		DialogAction{ID: "cancel", Label: "[Cancel]"},
	)
	d.SetFocus(true)

	if d.SelectedAction() != "allow" {
		t.Fatalf("initial action = %q, want %q", d.SelectedAction(), "allow")
	}

	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if d.SelectedAction() != "deny" {
		t.Fatalf("after first tab = %q, want %q", d.SelectedAction(), "deny")
	}

	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if d.SelectedAction() != "cancel" {
		t.Fatalf("after second tab = %q, want %q", d.SelectedAction(), "cancel")
	}

	// Wraps around.
	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if d.SelectedAction() != "allow" {
		t.Fatalf("after wrap tab = %q, want %q", d.SelectedAction(), "allow")
	}
}

func TestDialogPanel_EnterEmitsDialogResultMsg(t *testing.T) {
	d := NewDialogPanel("Title", "msg",
		DialogAction{ID: "allow", Label: "[Allow]"},
		DialogAction{ID: "deny", Label: "[Deny]"},
	)
	d.SetFocus(true)

	// Select "deny" first.
	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	result, ok := msg.(DialogResultMsg)
	if !ok {
		t.Fatalf("msg type = %T, want DialogResultMsg", msg)
	}
	if result.ActionID != "deny" {
		t.Fatalf("action = %q, want %q", result.ActionID, "deny")
	}
}

func TestDialogPanel_EscEmitsCancelResult(t *testing.T) {
	d := NewDialogPanel("Title", "msg",
		DialogAction{ID: "allow", Label: "[Allow]"},
	)
	d.SetFocus(true)

	_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a command")
	}
	msg := cmd()
	result, ok := msg.(DialogResultMsg)
	if !ok {
		t.Fatalf("msg type = %T, want DialogResultMsg", msg)
	}
	if result.ActionID != "cancel" {
		t.Fatalf("action = %q, want %q", result.ActionID, "cancel")
	}
}

func TestDialogPanel_SelectedActionReturnsCorrect(t *testing.T) {
	d := NewDialogPanel("Title", "msg",
		DialogAction{ID: "first", Label: "[First]"},
		DialogAction{ID: "second", Label: "[Second]"},
	)
	if d.SelectedAction() != "first" {
		t.Fatalf("initial = %q, want %q", d.SelectedAction(), "first")
	}
	d.SetFocus(true)
	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if d.SelectedAction() != "second" {
		t.Fatalf("after tab = %q, want %q", d.SelectedAction(), "second")
	}
}

func TestDialogPanel_NoActionsSelectedActionEmpty(t *testing.T) {
	d := NewDialogPanel("Title", "msg")
	if d.SelectedAction() != "" {
		t.Fatalf("no actions = %q, want empty", d.SelectedAction())
	}
}

func TestDialogPanel_UnfocusedIgnoresKeys(t *testing.T) {
	d := NewDialogPanel("Title", "msg",
		DialogAction{ID: "allow", Label: "[Allow]"},
		DialogAction{ID: "deny", Label: "[Deny]"},
	)
	// Not focused.
	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if d.SelectedAction() != "allow" {
		t.Fatalf("unfocused tab should not cycle: got %q", d.SelectedAction())
	}
}
