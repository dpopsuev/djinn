package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyRouter_HighPriorityConsumesFirst(t *testing.T) {
	r := NewKeyRouter()
	order := []string{}

	r.Register(KeyHandler{Name: "low", Priority: PriorityLow, Handle: func(msg tea.KeyMsg) (bool, tea.Cmd) {
		order = append(order, "low")
		return true, nil
	}})
	r.Register(KeyHandler{Name: "high", Priority: PriorityHigh, Handle: func(msg tea.KeyMsg) (bool, tea.Cmd) {
		order = append(order, "high")
		return true, nil
	}})

	consumed, _ := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !consumed {
		t.Fatal("should be consumed")
	}
	if len(order) != 1 || order[0] != "high" {
		t.Fatalf("order = %v, want [high] (low should be skipped)", order)
	}
}

func TestKeyRouter_PassthroughIfNotConsumed(t *testing.T) {
	r := NewKeyRouter()
	order := []string{}

	r.Register(KeyHandler{Name: "high", Priority: PriorityHigh, Handle: func(msg tea.KeyMsg) (bool, tea.Cmd) {
		order = append(order, "high")
		return false, nil // not consumed
	}})
	r.Register(KeyHandler{Name: "low", Priority: PriorityLow, Handle: func(msg tea.KeyMsg) (bool, tea.Cmd) {
		order = append(order, "low")
		return true, nil
	}})

	r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(order) != 2 || order[0] != "high" || order[1] != "low" {
		t.Fatalf("order = %v, want [high, low]", order)
	}
}

func TestKeyRouter_EmptyReturnsNotConsumed(t *testing.T) {
	r := NewKeyRouter()
	consumed, cmd := r.Handle(tea.KeyMsg{Type: tea.KeyEnter})
	if consumed || cmd != nil {
		t.Fatal("empty router should not consume")
	}
}

func TestKeyRouter_CriticalBeatsAll(t *testing.T) {
	r := NewKeyRouter()
	r.Register(KeyHandler{Name: "normal", Priority: PriorityNormal, Handle: func(msg tea.KeyMsg) (bool, tea.Cmd) {
		return true, nil
	}})
	r.Register(KeyHandler{Name: "critical", Priority: PriorityCritical, Handle: func(msg tea.KeyMsg) (bool, tea.Cmd) {
		return true, func() tea.Msg { return "critical" }
	}})

	_, cmd := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("critical handler should produce cmd")
	}
	msg := cmd()
	if msg != "critical" {
		t.Fatalf("got %v, want critical", msg)
	}
}
