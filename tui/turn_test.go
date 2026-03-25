package tui

import (
	"strings"
	"testing"
)

func TestTurnPanel_SummaryView(t *testing.T) {
	tp := NewTurnPanel("t1", "fix the bug", "Fixed it.", "", nil, 100, 50)
	tp.Toggle() // collapse
	view := tp.View(80)
	if !strings.Contains(view, "fix the bug") {
		t.Fatalf("summary should show prompt: %q", view)
	}
	if !strings.Contains(view, "150 tok") {
		t.Fatalf("summary should show token count: %q", view)
	}
}

func TestTurnPanel_ExpandedView(t *testing.T) {
	tools := []Panel{NewEnvelopePanel("e1", "Read", "main.go")}
	tp := NewTurnPanel("t1", "fix the bug", "Fixed the nil pointer.", "analyzing code", tools, 100, 50)
	view := tp.View(80)

	// User prompt at top.
	if !strings.Contains(view, "fix the bug") {
		t.Fatal("expanded should show user prompt")
	}
	// Tool call in middle.
	if !strings.Contains(view, "Read") {
		t.Fatal("expanded should show tool call")
	}
	// Agent text below.
	if !strings.Contains(view, "nil pointer") {
		t.Fatal("expanded should show agent text")
	}
	// Thinking at bottom.
	if !strings.Contains(view, "analyzing code") {
		t.Fatal("expanded should show thinking")
	}
}

func TestTurnPanel_Children_ReturnsToolCalls(t *testing.T) {
	tools := []Panel{
		NewEnvelopePanel("e1", "Read", "a.go"),
		NewEnvelopePanel("e2", "Write", "b.go"),
	}
	tp := NewTurnPanel("t1", "prompt", "response", "", tools, 0, 0)
	children := tp.Children()
	if len(children) != 2 {
		t.Fatalf("children = %d, want 2", len(children))
	}
}

func TestTurnPanel_Collapsible(t *testing.T) {
	tp := NewTurnPanel("t1", "prompt", "response", "", nil, 0, 0)
	if !tp.Collapsible() {
		t.Fatal("TurnPanel should be collapsible")
	}
}
