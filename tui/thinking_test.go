package tui

import (
	"strings"
	"testing"
)

func TestThinkingPanel_ShowAndClear(t *testing.T) {
	p := NewThinkingPanel()
	p.Update(ThinkingMsg("analyzing code"))
	if !p.Active() {
		t.Fatal("should be active after ThinkingMsg")
	}
	view := p.View(80)
	if !strings.Contains(view, "analyzing code") {
		t.Fatalf("view = %q", view)
	}

	p.Update(ThinkingClearMsg{})
	if p.Active() {
		t.Fatal("should be inactive after clear")
	}
	if p.View(80) != "" {
		t.Fatal("cleared view should be empty")
	}
}

func TestThinkingPanel_ViewEmpty(t *testing.T) {
	p := NewThinkingPanel()
	if p.View(80) != "" {
		t.Fatal("initial view should be empty")
	}
}
