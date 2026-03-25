// list.go — ListPanel is a generic drillable list.
// Each item is a Panel. Cursor selects. Enter=Dive into Children().
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ListSetItemsMsg replaces the list items.
type ListSetItemsMsg struct{ Items []Panel }

// ListPanel is a selectable list of Panels. Cursor-based navigation.
// Children() returns the selected item's children for Dive/Climb.
type ListPanel struct {
	BasePanel
	items      []Panel
	cursor     int
	scrollOff  int
	visibleMax int // max items visible (0 = all)
}

var _ Panel = (*ListPanel)(nil)

// NewListPanel creates a list with the given ID and items.
func NewListPanel(id string, items []Panel) *ListPanel {
	return &ListPanel{
		BasePanel:  NewBasePanel(id, 0),
		items:      items,
		visibleMax: 20,
	}
}

// Children returns the selected item's children — enables Dive.
func (p *ListPanel) Children() []Panel {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		return p.items[p.cursor].Children()
	}
	return nil
}

// Items returns all items.
func (p *ListPanel) Items() []Panel { return p.items }

// SetItems replaces the item list.
func (p *ListPanel) SetItems(items []Panel) {
	p.items = items
	if p.cursor >= len(items) {
		p.cursor = max(0, len(items)-1)
	}
	p.scrollOff = 0
}

// Cursor returns the current cursor position.
func (p *ListPanel) Cursor() int { return p.cursor }

// Selected returns the currently highlighted panel (or nil).
func (p *ListPanel) Selected() Panel {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		return p.items[p.cursor]
	}
	return nil
}

func (p *ListPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case ListSetItemsMsg:
		p.SetItems(msg.Items)
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch msg.Type {
		case tea.KeyUp:
			if p.cursor > 0 {
				p.cursor--
				p.ensureVisible()
			}
		case tea.KeyDown:
			if p.cursor < len(p.items)-1 {
				p.cursor++
				p.ensureVisible()
			}
		}
	}
	return p, nil
}

func (p *ListPanel) ensureVisible() {
	if p.visibleMax <= 0 {
		return
	}
	if p.cursor < p.scrollOff {
		p.scrollOff = p.cursor
	}
	if p.cursor >= p.scrollOff+p.visibleMax {
		p.scrollOff = p.cursor - p.visibleMax + 1
	}
}

func (p *ListPanel) View(width int) string {
	if len(p.items) == 0 {
		return DimStyle.Render("  (empty)")
	}

	start, end := 0, len(p.items)
	if p.visibleMax > 0 {
		start = p.scrollOff
		end = min(p.scrollOff+p.visibleMax, len(p.items))
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		if i > start {
			sb.WriteByte('\n')
		}
		prefix := "  "
		if i == p.cursor {
			prefix = ActiveGlyphs.ListCursor + " "
		}
		sb.WriteString(prefix + p.items[i].View(width-2))
	}
	return sb.String()
}

