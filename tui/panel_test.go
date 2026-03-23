package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

// --- Focus depth dimming tests ---

func TestFocusDepths_MiddleFocused(t *testing.T) {
	depths := FocusDepths(5, 2)
	expected := []int{2, 1, 0, 1, 2}
	for i, d := range depths {
		if d != expected[i] {
			t.Fatalf("depth[%d] = %d, want %d", i, d, expected[i])
		}
	}
}

func TestFocusDepths_FirstFocused(t *testing.T) {
	depths := FocusDepths(3, 0)
	if depths[0] != 0 || depths[1] != 1 || depths[2] != 2 {
		t.Fatalf("depths = %v", depths)
	}
}

func TestRenderWithDepth_Focused(t *testing.T) {
	result := RenderWithDepth("hello", 0)
	if result != "hello" {
		t.Fatal("depth 0 should return content unchanged")
	}
}

func TestRenderWithDepth_Unfocused_DiffersFromFocused(t *testing.T) {
	// Force color output — lipgloss emits nothing without a TTY
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	focused := RenderWithDepth("hello", 0)
	dim1 := RenderWithDepth("hello", 1)
	dim2 := RenderWithDepth("hello", 2)

	if dim1 == "" || dim2 == "" {
		t.Fatal("dimmed output should not be empty")
	}
	if dim1 == focused {
		t.Fatal("depth 1 MUST differ from depth 0 — dimming has no visible effect")
	}
	if dim2 == focused {
		t.Fatal("depth 2 MUST differ from depth 0")
	}
}

func TestRenderFocusIndicator(t *testing.T) {
	focused := RenderFocusIndicator(true)
	unfocused := RenderFocusIndicator(false)
	if focused == unfocused {
		t.Fatal("focused indicator must differ from unfocused")
	}
	if focused == "" {
		t.Fatal("focused indicator should not be empty")
	}
}

func TestInputPanel_TabComplete_Prefix(t *testing.T) {
	p := NewInputPanel()
	p.SetCompletions([]string{"/clear", "/compact", "/config", "/config-save", "/help"})

	p.SetValue("/co")
	if !p.TabComplete() {
		t.Fatal("should handle /co prefix")
	}
	val := p.Value()
	if val != "/compact" && val != "/config" && val != "/config-save" {
		t.Fatalf("completed = %q, want one of /compact, /config, /config-save", val)
	}
}

func TestInputPanel_TabComplete_NoSlash(t *testing.T) {
	p := NewInputPanel()
	p.SetCompletions([]string{"/help"})

	p.SetValue("hello")
	if p.TabComplete() {
		t.Fatal("should not handle non-slash input")
	}
}

func TestInputPanel_TabComplete_Cycle(t *testing.T) {
	p := NewInputPanel()
	p.SetCompletions([]string{"/config", "/config-save", "/help"})

	p.SetValue("/config")
	p.TabComplete()
	first := p.Value()

	p.TabComplete() // cycle
	second := p.Value()

	if first == second {
		t.Fatal("cycling should produce different value")
	}
}

func TestInputPanel_TabComplete_ExactMatch(t *testing.T) {
	p := NewInputPanel()
	p.SetCompletions([]string{"/help", "/history"})

	p.SetValue("/help")
	if !p.TabComplete() {
		t.Fatal("should handle exact match")
	}
	if p.Value() != "/help" {
		t.Fatalf("exact match should complete to /help, got %q", p.Value())
	}
}

func TestInputPanel_TabComplete_NoMatch(t *testing.T) {
	p := NewInputPanel()
	p.SetCompletions([]string{"/help", "/config"})

	p.SetValue("/zzz")
	if !p.TabComplete() {
		t.Fatal("should consume Tab even with no matches")
	}
	if p.Value() != "/zzz" {
		t.Fatalf("no match should keep original value, got %q", p.Value())
	}
}

func TestInputPanel_Visible(t *testing.T) {
	p := NewInputPanel()
	if !p.Visible() {
		t.Fatal("should be visible by default")
	}
	p.SetVisible(false)
	if p.Visible() {
		t.Fatal("should be hidden after SetVisible(false)")
	}
	if p.View(80) != "" {
		t.Fatal("hidden panel should return empty view")
	}
}

func TestOutputPanel_Overlay(t *testing.T) {
	p := NewOutputPanel()
	p.Append("hello")
	p.SetOverlay("thinking...")

	view := p.View(80)
	if !strings.Contains(view, "hello") {
		t.Fatal("should contain lines")
	}
	if !strings.Contains(view, "thinking...") {
		t.Fatal("should contain overlay")
	}

	p.SetOverlay("")
	view = p.View(80)
	if strings.Contains(view, "thinking...") {
		t.Fatal("overlay should be cleared")
	}
}

func TestOutputPanel_AppendToLast(t *testing.T) {
	p := NewOutputPanel()
	p.Append("hello ")
	p.AppendToLast("world")

	if p.Lines()[0] != "hello world" {
		t.Fatalf("line = %q, want 'hello world'", p.Lines()[0])
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
