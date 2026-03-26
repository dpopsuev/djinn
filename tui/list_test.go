package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func makeItems(names ...string) []Panel {
	items := make([]Panel, len(names))
	for i, n := range names {
		items[i] = NewEnvelopePanel(n, n, "")
	}
	return items
}

func TestListPanel_CursorNavigation(t *testing.T) {
	p := NewListPanel("test", makeItems("a", "b", "c"))
	p.SetFocus(true)
	if p.Cursor() != 0 {
		t.Fatalf("initial cursor = %d", p.Cursor())
	}

	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Cursor() != 1 {
		t.Fatalf("after down cursor = %d", p.Cursor())
	}

	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.Cursor() != 0 {
		t.Fatalf("after up cursor = %d", p.Cursor())
	}

	// Can't go above 0.
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.Cursor() != 0 {
		t.Fatal("cursor should not go below 0")
	}
}

func TestListPanel_CursorBounds(t *testing.T) {
	p := NewListPanel("test", makeItems("a", "b"))
	p.SetFocus(true)
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.Update(tea.KeyMsg{Type: tea.KeyDown}) // beyond end
	if p.Cursor() != 1 {
		t.Fatalf("cursor = %d, should stop at last item", p.Cursor())
	}
}

func TestListPanel_Children_ReturnsSelectedChildren(t *testing.T) {
	// EnvelopePanel has no children — returns nil.
	p := NewListPanel("test", makeItems("a", "b"))
	if p.Children() != nil {
		t.Fatal("envelope has no children")
	}
}

func TestListPanel_Selected(t *testing.T) {
	p := NewListPanel("test", makeItems("a", "b", "c"))
	sel := p.Selected()
	if sel == nil || sel.ID() != "a" {
		t.Fatalf("selected = %v", sel)
	}

	p.SetFocus(true)
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	sel = p.Selected()
	if sel == nil || sel.ID() != "b" {
		t.Fatalf("selected after down = %v", sel)
	}
}

func TestListPanel_SetItems(t *testing.T) {
	p := NewListPanel("test", makeItems("a", "b", "c"))
	p.SetFocus(true)
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.Update(tea.KeyMsg{Type: tea.KeyDown}) // cursor at 2

	p.SetItems(makeItems("x")) // only 1 item
	if p.Cursor() != 0 {
		t.Fatalf("cursor = %d, should reset to 0 with fewer items", p.Cursor())
	}
}

func TestListPanel_EmptyView(t *testing.T) {
	p := NewListPanel("test", nil)
	view := p.View(80)
	if !strings.Contains(view, "empty") {
		t.Fatalf("empty list should show (empty): %q", view)
	}
}

func TestListPanel_View_ShowsCursor(t *testing.T) {
	p := NewListPanel("test", makeItems("alpha", "beta"))
	view := p.View(80)
	// First item should have cursor marker.
	if !strings.Contains(view, ActiveGlyphs.ListCursor) {
		t.Fatalf("view should show cursor glyph %q: %q", ActiveGlyphs.ListCursor, view)
	}
}

func TestListPanel_ScrollsOnOverflow(t *testing.T) {
	items := makeItems("a", "b", "c", "d", "e")
	p := NewListPanel("test", items)
	p.visibleMax = 3
	p.SetFocus(true)

	// Move cursor to item 4 (index 3).
	for range 3 {
		p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	view := p.View(80)
	// Should show items around cursor, not first 3.
	if strings.Contains(view, "alpha") {
		t.Fatal("scrolled view should not show first item")
	}
}

func TestListPanel_UnfocusedIgnoresKeys(t *testing.T) {
	p := NewListPanel("test", makeItems("a", "b"))
	// Not focused — keys should be ignored.
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Cursor() != 0 {
		t.Fatal("unfocused should not respond to keys")
	}
}
