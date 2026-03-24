package tui

import (
	"strings"
	"testing"
)

// --- Isolated OutputPanel tests: zero imports from cli/repl, agent, driver, session ---

func TestOutputPanel_AppendViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(ResizeMsg{80, 20})
	p.Update(OutputAppendMsg{"hello"})
	p.Update(OutputAppendMsg{"world"})
	if p.LineCount() != 2 {
		t.Fatalf("lines = %d, want 2", p.LineCount())
	}
	view := p.View(80)
	if !strings.Contains(view, "hello") || !strings.Contains(view, "world") {
		t.Fatalf("view missing content: %q", view)
	}
}

func TestOutputPanel_SetLineViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(OutputAppendMsg{"original"})
	p.Update(OutputSetLineMsg{0, "replaced"})
	if p.Lines()[0] != "replaced" {
		t.Fatalf("line = %q", p.Lines()[0])
	}
}

func TestOutputPanel_AppendLastViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(OutputAppendMsg{"hello "})
	p.Update(OutputAppendLastMsg{"world"})
	if p.Lines()[0] != "hello world" {
		t.Fatalf("line = %q", p.Lines()[0])
	}
}

func TestOutputPanel_ClearViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(OutputAppendMsg{"line1"})
	p.Update(OutputAppendMsg{"line2"})
	p.Update(OutputClearMsg{})
	if p.LineCount() != 0 {
		t.Fatalf("lines = %d after clear", p.LineCount())
	}
}

func TestOutputPanel_OverlayViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(ResizeMsg{80, 20})
	p.Update(OutputAppendMsg{"content"})
	p.Update(OutputSetOverlayMsg{"thinking..."})
	view := p.View(80)
	if !strings.Contains(view, "thinking...") {
		t.Fatal("overlay should appear in view")
	}
	p.Update(OutputSetOverlayMsg{""})
	view = p.View(80)
	if strings.Contains(view, "thinking...") {
		t.Fatal("overlay should be cleared")
	}
}

func TestOutputPanel_StreamViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(ResizeMsg{80, 20})
	p.Update(OutputAppendMsg{"prefix: "})

	// Stream tokens via TextMsg
	p.Update(TextMsg("hello "))
	p.Update(TextMsg("world"))

	// Flush
	p.Update(FlushStreamMsg{})

	last := p.Lines()[p.LineCount()-1]
	if !strings.Contains(last, "hello world") {
		t.Fatalf("last line after flush = %q, want 'hello world'", last)
	}
}

func TestOutputPanel_StreamFlush_Empty(t *testing.T) {
	p := NewOutputPanel()
	before := p.LineCount()
	p.Update(FlushStreamMsg{})
	if p.LineCount() != before {
		t.Fatal("empty flush should not modify lines")
	}
}

func TestOutputPanel_ResizeViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(ResizeMsg{80, 20})
	p.Update(OutputAppendMsg{"test"})
	view := p.View(80)
	if !strings.Contains(view, "test") {
		t.Fatal("should render after resize")
	}
}

func TestOutputPanel_DirtyFlag(t *testing.T) {
	p := NewOutputPanel()
	p.Update(ResizeMsg{80, 20})
	p.Update(OutputAppendMsg{"line"})

	// First View() syncs viewport
	v1 := p.View(80)
	// Second View() without mutations — should return same
	v2 := p.View(80)
	if v1 != v2 {
		t.Fatal("consecutive views without mutations should match")
	}
}
