package tui

import (
	"strings"
	"testing"
)

func TestTurnEnvelope_NewCreatesCorrectly(t *testing.T) {
	te := NewTurnEnvelope(1, "fix the bug")
	if te.turnIdx != 1 {
		t.Fatalf("turnIdx = %d, want 1", te.turnIdx)
	}
	if te.userInput != "fix the bug" {
		t.Fatalf("userInput = %q, want %q", te.userInput, "fix the bug")
	}
	if te.complete {
		t.Fatal("should not be complete on creation")
	}
	if te.Collapsed() {
		t.Fatal("should start expanded")
	}
	if te.ID() != "turn-1" {
		t.Fatalf("ID = %q, want %q", te.ID(), "turn-1")
	}
}

func TestTurnEnvelope_AddTextAccumulates(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	te.AddText("hello ")
	te.AddText("world")
	if got := te.response.String(); got != "hello world" {
		t.Fatalf("response = %q, want %q", got, "hello world")
	}
}

func TestTurnEnvelope_AddToolAppendsToSlice(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	env1 := NewEnvelopePanel("e1", "Read", "main.go")
	env2 := NewEnvelopePanel("e2", "Write", "out.go")
	te.AddTool(env1)
	te.AddTool(env2)
	if len(te.tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(te.tools))
	}
}

func TestTurnEnvelope_CompleteSetsFlag(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	te.Complete()
	if !te.complete {
		t.Fatal("complete should be true after Complete()")
	}
}

func TestTurnEnvelope_ViewContainsUserInput(t *testing.T) {
	te := NewTurnEnvelope(1, "refactor the parser")
	view := te.View(80)
	if !strings.Contains(view, "refactor the parser") {
		t.Fatalf("view should contain user input: %q", view)
	}
	if !strings.Contains(view, "Turn #1") {
		t.Fatalf("view should contain turn title: %q", view)
	}
}

func TestTurnEnvelope_ViewContainsResponse(t *testing.T) {
	te := NewTurnEnvelope(2, "prompt")
	te.AddText("I fixed the parser")
	view := te.View(80)
	if !strings.Contains(view, "I fixed the parser") {
		t.Fatalf("view should contain response: %q", view)
	}
}

func TestTurnEnvelope_ViewContainsToolSummaries(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	env := NewEnvelopePanel("e1", "Read", "main.go")
	te.AddTool(env)
	view := te.View(80)
	if !strings.Contains(view, "Read") {
		t.Fatalf("view should contain tool name: %q", view)
	}
}

func TestTurnEnvelope_ViewContainsThinking(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	te.AddThinking("analyzing the code")
	view := te.View(80)
	if !strings.Contains(view, "analyzing the code") {
		t.Fatalf("view should contain thinking: %q", view)
	}
}

func TestTurnEnvelope_ViewOmitsEmptyThinking(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	view := te.View(80)
	// Should not contain spinner glyph when thinking is empty.
	if strings.Contains(view, SpinnerFrames[0]+" ") {
		t.Fatalf("view should not contain spinner when thinking is empty: %q", view)
	}
}

func TestTurnEnvelope_CollapsedViewShowsSingleLine(t *testing.T) {
	te := NewTurnEnvelope(3, "fix the null pointer dereference")
	te.Toggle() // collapse
	view := te.View(80)
	if !strings.Contains(view, "Turn #3") {
		t.Fatalf("collapsed should show turn number: %q", view)
	}
	if !strings.Contains(view, "fix the null pointer") {
		t.Fatalf("collapsed should show truncated input: %q", view)
	}
	// Collapsed view should be a single logical line (no border box).
	lines := strings.Split(view, "\n")
	if len(lines) != 1 {
		t.Fatalf("collapsed view should be single line, got %d lines", len(lines))
	}
}

func TestTurnEnvelope_CollapsedTruncatesLongInput(t *testing.T) {
	long := strings.Repeat("a", 100)
	te := NewTurnEnvelope(1, long)
	te.Toggle()
	view := te.View(80)
	if len(view) > 200 { // account for ANSI codes from DimStyle
		t.Fatalf("collapsed view too long: %d chars", len(view))
	}
	if !strings.Contains(view, "...") {
		t.Fatalf("collapsed should truncate with ellipsis: %q", view)
	}
}

func TestTurnEnvelope_ChildrenReturnsToolEnvelopes(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	env1 := NewEnvelopePanel("e1", "Read", "a.go")
	env2 := NewEnvelopePanel("e2", "Write", "b.go")
	te.AddTool(env1)
	te.AddTool(env2)
	children := te.Children()
	if len(children) != 2 {
		t.Fatalf("children = %d, want 2", len(children))
	}
	if children[0].ID() != "e1" {
		t.Fatalf("first child ID = %q, want %q", children[0].ID(), "e1")
	}
	if children[1].ID() != "e2" {
		t.Fatalf("second child ID = %q, want %q", children[1].ID(), "e2")
	}
}

func TestTurnEnvelope_ChildrenEmptyWhenNoTools(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	children := te.Children()
	if len(children) != 0 {
		t.Fatalf("children = %d, want 0", len(children))
	}
}

func TestTurnEnvelope_ToggleSwitchesState(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	if te.Collapsed() {
		t.Fatal("should start expanded")
	}
	te.Toggle()
	if !te.Collapsed() {
		t.Fatal("should be collapsed after first toggle")
	}
	te.Toggle()
	if te.Collapsed() {
		t.Fatal("should be expanded after second toggle")
	}
}

func TestTurnEnvelope_Collapsible(t *testing.T) {
	te := NewTurnEnvelope(1, "prompt")
	if !te.Collapsible() {
		t.Fatal("TurnEnvelope should be collapsible")
	}
}

func TestTurnEnvelope_ExpandedHasBorder(t *testing.T) {
	te := NewTurnEnvelope(1, "hello")
	view := te.View(80)
	// Rounded border uses unicode box chars — check for corner pieces.
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Fatalf("expanded view should have rounded border chars: %q", view)
	}
}
