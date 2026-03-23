package tui

import (
	"strings"
	"testing"
)

func TestFocusManager_Cycle(t *testing.T) {
	p1 := NewOutputPanel()
	p2 := NewDashboardPanel()
	p3 := NewOutputPanel()
	p3.BasePanel = NewBasePanel("output2", 0)

	fm := NewFocusManager(p1, p2, p3)

	if fm.Active().ID() != "output" {
		t.Fatalf("initial focus should be output, got %s", fm.Active().ID())
	}

	fm.Cycle()
	if fm.Active().ID() != "dashboard" {
		t.Fatalf("after cycle should be dashboard, got %s", fm.Active().ID())
	}

	fm.Cycle()
	if fm.Active().ID() != "output2" {
		t.Fatalf("after 2 cycles should be output2, got %s", fm.Active().ID())
	}

	fm.Cycle()
	if fm.Active().ID() != "output" {
		t.Fatal("should wrap around to output")
	}
}

func TestFocusManager_FocusUp(t *testing.T) {
	p1 := NewOutputPanel()
	p2 := NewDashboardPanel()
	fm := NewFocusManager(p1, p2)

	fm.FocusUp()
	if fm.Active().ID() != "dashboard" {
		t.Fatal("FocusUp from first should wrap to last")
	}
}

func TestFocusManager_SetFocusUpdatesPanel(t *testing.T) {
	p1 := NewOutputPanel()
	p2 := NewDashboardPanel()
	fm := NewFocusManager(p1, p2)

	if !p1.Focused() {
		t.Fatal("output should be focused")
	}
	if p2.Focused() {
		t.Fatal("dashboard should not be focused")
	}

	fm.Cycle()
	if p1.Focused() {
		t.Fatal("output should lose focus after cycle")
	}
	if !p2.Focused() {
		t.Fatal("dashboard should gain focus after cycle")
	}
}

func TestOutputPanel_AppendAndView(t *testing.T) {
	p := NewOutputPanel()
	p.Append("hello")
	p.Append("world")

	if p.LineCount() != 2 {
		t.Fatalf("lines = %d", p.LineCount())
	}

	view := p.View(80)
	if !strings.Contains(view, "hello") || !strings.Contains(view, "world") {
		t.Fatalf("view = %q", view)
	}
}

func TestOutputPanel_SetLine(t *testing.T) {
	p := NewOutputPanel()
	p.Append("original")
	p.SetLine(0, "replaced")

	if p.Lines()[0] != "replaced" {
		t.Fatalf("line = %q", p.Lines()[0])
	}
}

func TestEnvelopePanel_CollapsedView(t *testing.T) {
	e := NewEnvelopePanel("e1", "Read", "test.go")
	e.SetResult("file contents\nline 2\nline 3", false)

	if !e.Collapsed() {
		t.Fatal("should auto-collapse after result")
	}

	view := e.View(80)
	if !strings.Contains(view, "Read") {
		t.Fatalf("collapsed should show tool name: %q", view)
	}
}

func TestEnvelopePanel_Toggle(t *testing.T) {
	e := NewEnvelopePanel("e1", "Read", "test.go")
	e.SetResult("output", false)

	if !e.Collapsed() {
		t.Fatal("should start collapsed")
	}
	e.Toggle()
	if e.Collapsed() {
		t.Fatal("should be expanded after toggle")
	}
	e.Toggle()
	if !e.Collapsed() {
		t.Fatal("should be collapsed after second toggle")
	}
}

func TestEnvelopePanel_Collapsible(t *testing.T) {
	e := NewEnvelopePanel("e1", "Read", "")
	if !e.Collapsible() {
		t.Fatal("envelopes should be collapsible")
	}
}

func TestDashboardPanel_View(t *testing.T) {
	d := NewDashboardPanel()
	d.SetIdentity("aeon", "claude", "opus", "agent")
	d.SetMetrics(100, 50, 3)

	view := d.View(80)
	if view == "" {
		t.Fatal("dashboard should produce output")
	}
}

func TestSeparator_Depths(t *testing.T) {
	s0 := Separator(20, 0, false)
	s1 := Separator(20, 1, false)
	s2 := Separator(20, 2, false)
	s3 := Separator(20, 3, false)

	// Each should produce non-empty output
	for i, s := range []string{s0, s1, s2, s3} {
		if s == "" {
			t.Fatalf("depth %d separator should not be empty", i)
		}
	}
}

func TestSeparator_ZeroWidth(t *testing.T) {
	s := Separator(0, 0, false)
	if s != "" {
		t.Fatal("zero width should return empty")
	}
}

func TestSeparator_Focused(t *testing.T) {
	unfocused := Separator(10, 0, false)
	focused := Separator(10, 0, true)

	// Both should produce output, but different (focused has color)
	if unfocused == "" || focused == "" {
		t.Fatal("both should produce output")
	}
}

func TestInputPanel_History(t *testing.T) {
	p := NewInputPanel()
	p.AddHistory("first")
	p.AddHistory("second")

	p.HistoryUp()
	if p.Value() != "second" {
		t.Fatalf("up = %q, want second", p.Value())
	}

	p.HistoryUp()
	if p.Value() != "first" {
		t.Fatalf("up again = %q, want first", p.Value())
	}

	p.HistoryDown()
	if p.Value() != "second" {
		t.Fatalf("down = %q, want second", p.Value())
	}
}

func TestBasePanel_Defaults(t *testing.T) {
	b := NewBasePanel("test", 5)
	if b.ID() != "test" {
		t.Fatalf("id = %q", b.ID())
	}
	if b.Height() != 5 {
		t.Fatalf("height = %d", b.Height())
	}
	if b.Focused() {
		t.Fatal("should not be focused by default")
	}
	if b.Collapsible() {
		t.Fatal("base panel should not be collapsible")
	}
	if b.Children() != nil {
		t.Fatal("base panel should have no children")
	}
}
