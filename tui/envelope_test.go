package tui

import (
	"strings"
	"testing"
)

// --- Isolated EnvelopePanel tests: zero domain imports ---

func TestEnvelope_ToolResultViaMessage(t *testing.T) {
	e := NewEnvelopePanel("e1", "Read", "test.go")
	e.Update(ToolResultMsg{Name: "Read", Output: "file contents\nline 2", IsError: false})
	if !e.Collapsed() {
		t.Fatal("should auto-collapse on result")
	}
	view := e.View(80)
	if !strings.Contains(view, "Read") {
		t.Fatalf("collapsed view should show tool name: %q", view)
	}
}

func TestEnvelope_ErrorResult(t *testing.T) {
	e := NewEnvelopePanel("e2", "Bash", "ls")
	e.Update(ToolResultMsg{Name: "Bash", Output: "permission denied", IsError: true})
	view := e.View(80)
	if !strings.Contains(view, "Bash") {
		t.Fatal("should show tool name")
	}
}

func TestEnvelope_ExpandedBeforeResult(t *testing.T) {
	e := NewEnvelopePanel("e3", "Write", `{"path":"foo.go"}`)
	if e.Collapsed() {
		t.Fatal("should start expanded (no result yet)")
	}
	view := e.View(80)
	if !strings.Contains(view, "Write") {
		t.Fatal("expanded view should show tool name")
	}
	if !strings.Contains(view, "foo.go") {
		t.Fatal("expanded view should show args")
	}
}

func TestEnvelope_ToggleAfterResult(t *testing.T) {
	e := NewEnvelopePanel("e4", "Read", "x.go")
	e.Update(ToolResultMsg{Name: "Read", Output: "content"})
	if !e.Collapsed() {
		t.Fatal("should be collapsed after result")
	}
	e.Toggle()
	if e.Collapsed() {
		t.Fatal("should be expanded after toggle")
	}
	view := e.View(80)
	if !strings.Contains(view, "content") {
		t.Fatal("expanded should show output")
	}
}
