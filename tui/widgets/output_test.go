package widgets

import (
	"strings"
	"testing"

	tui "github.com/dpopsuev/djinn/tui"
)

// --- Isolated OutputPanel tests: zero imports from cli/repl, agent, driver, session ---

func TestOutputPanel_AppendViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.ResizeMsg{Width: 80, Height: 20})
	p.Update(tui.OutputAppendMsg{Line: "hello"})
	p.Update(tui.OutputAppendMsg{Line: "world"})
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
	p.Update(tui.OutputAppendMsg{Line: "original"})
	p.Update(tui.OutputSetLineMsg{Index: 0, Line: "replaced"})
	if p.Lines()[0] != "replaced" {
		t.Fatalf("line = %q", p.Lines()[0])
	}
}

func TestOutputPanel_AppendLastViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.OutputAppendMsg{Line: "hello "})
	p.Update(tui.OutputAppendLastMsg{Text: "world"})
	if p.Lines()[0] != "hello world" {
		t.Fatalf("line = %q", p.Lines()[0])
	}
}

func TestOutputPanel_ClearViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.OutputAppendMsg{Line: "line1"})
	p.Update(tui.OutputAppendMsg{Line: "line2"})
	p.Update(tui.OutputClearMsg{})
	if p.LineCount() != 0 {
		t.Fatalf("lines = %d after clear", p.LineCount())
	}
}

func TestOutputPanel_OverlayViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.ResizeMsg{Width: 80, Height: 20})
	p.Update(tui.OutputAppendMsg{Line: "content"})
	p.Update(tui.OutputSetOverlayMsg{Text: "thinking..."})
	view := p.View(80)
	if !strings.Contains(view, "thinking...") {
		t.Fatal("overlay should appear in view")
	}
	p.Update(tui.OutputSetOverlayMsg{Text: ""})
	view = p.View(80)
	if strings.Contains(view, "thinking...") {
		t.Fatal("overlay should be cleared")
	}
}

func TestOutputPanel_StreamViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.ResizeMsg{Width: 80, Height: 20})
	p.Update(tui.OutputAppendMsg{Line: "prefix: "})

	// Stream tokens via TextMsg
	p.Update(tui.TextMsg("hello "))
	p.Update(tui.TextMsg("world"))

	// Flush
	p.Update(tui.FlushStreamMsg{})

	last := p.Lines()[p.LineCount()-1]
	if !strings.Contains(last, "hello world") {
		t.Fatalf("last line after flush = %q, want 'hello world'", last)
	}
}

func TestOutputPanel_StreamFlush_Empty(t *testing.T) {
	p := NewOutputPanel()
	before := p.LineCount()
	p.Update(tui.FlushStreamMsg{})
	if p.LineCount() != before {
		t.Fatal("empty flush should not modify lines")
	}
}

func TestOutputPanel_ResizeViaMessage(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.ResizeMsg{Width: 80, Height: 20})
	p.Update(tui.OutputAppendMsg{Line: "test"})
	view := p.View(80)
	if !strings.Contains(view, "test") {
		t.Fatal("should render after resize")
	}
}

func TestOutputPanel_DirtyFlag(t *testing.T) {
	p := NewOutputPanel()
	p.Update(tui.ResizeMsg{Width: 80, Height: 20})
	p.Update(tui.OutputAppendMsg{Line: "line"})

	// First View() syncs viewport
	v1 := p.View(80)
	// Second View() without mutations — should return same
	v2 := p.View(80)
	if v1 != v2 {
		t.Fatal("consecutive views without mutations should match")
	}
}
