package widgets

import (
	"strings"
	"testing"

	tui "github.com/dpopsuev/djinn/tui"
)

func TestThinkingPanel_ShowAndClear(t *testing.T) {
	p := NewThinkingPanel()
	p.Update(tui.ThinkingMsg("analyzing code"))
	if !p.Active() {
		t.Fatal("should be active after ThinkingMsg")
	}
	view := p.View(80)
	if !strings.Contains(view, "analyzing code") {
		t.Fatalf("view = %q", view)
	}

	p.Update(tui.ThinkingClearMsg{})
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
