package tui

import (
	"strings"
	"testing"
)

func TestCommandsPanel_ShowAndFilter(t *testing.T) {
	p := NewCommandsPanel([]string{"/help", "/mode", "/config", "/clear"})
	p.Update(CommandsShowMsg{Filter: "/c"})

	if !p.Active() {
		t.Fatal("should be active after show")
	}
	view := p.View(80)
	if !strings.Contains(view, "/config") || !strings.Contains(view, "/clear") {
		t.Fatalf("view should show /c matches: %q", view)
	}
	if strings.Contains(view, "/help") {
		t.Fatal("view should NOT show /help (doesn't match /c)")
	}
}

func TestCommandsPanel_Hide(t *testing.T) {
	p := NewCommandsPanel([]string{"/help"})
	p.Update(CommandsShowMsg{Filter: "/"})
	p.Update(CommandsHideMsg{})

	if p.Active() {
		t.Fatal("should be inactive after hide")
	}
	if p.View(80) != "" {
		t.Fatal("hidden view should be empty")
	}
}

func TestCommandsPanel_Selected(t *testing.T) {
	p := NewCommandsPanel([]string{"/help", "/mode", "/config"})
	p.Update(CommandsShowMsg{Filter: "/"})

	if p.Selected() != "/help" {
		t.Fatalf("selected = %q, want /help", p.Selected())
	}
}

func TestCommandsPanel_EmptyFilter(t *testing.T) {
	p := NewCommandsPanel([]string{"/help"})
	p.Update(CommandsShowMsg{Filter: "/zzz"})

	if p.Active() {
		t.Fatal("no matches should not be active")
	}
}
