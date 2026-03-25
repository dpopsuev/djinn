// layout_engine.go — declarative panel composition, separate from panel logic.
// Panels register with rules. Engine computes visibility, allocates height,
// renders borders, composes final view. model.go View() becomes one call.
package tui

import "strings"

// BorderMode controls panel border rendering.
type BorderMode int

const (
	BorderFocusDepth BorderMode = iota // focused=red, unfocused=dim
	BorderOnly                         // border without content dimming (output panel)
	BorderNone                         // no border
)

// PanelSlot registers a panel with visibility and layout rules.
type PanelSlot struct {
	Panel     Panel
	Visible   func() bool // nil = always visible
	Weight    int         // 0 = fixed height (uses Panel.Height()), >0 = flex
	MinHeight int         // minimum height for flex panels
	Border    BorderMode
	Focusable bool // included in focus cycling
}

// LayoutEngine computes panel positions and renders the final view.
type LayoutEngine struct {
	slots  []PanelSlot
	focus  *FocusManager
	width  int
	height int
}

// NewLayoutEngine creates an engine with the given focus manager.
func NewLayoutEngine(fm *FocusManager) *LayoutEngine {
	return &LayoutEngine{focus: fm}
}

// Register adds a panel slot to the layout.
func (e *LayoutEngine) Register(slot PanelSlot) {
	e.slots = append(e.slots, slot)
}

// Resize updates terminal dimensions.
func (e *LayoutEngine) Resize(width, height int) {
	e.width = width
	e.height = height
}

// VisibleSlots returns slots where Visible() == true (or Visible is nil).
func (e *LayoutEngine) VisibleSlots() []PanelSlot {
	var out []PanelSlot
	for _, s := range e.slots {
		if s.Visible == nil || s.Visible() {
			out = append(out, s)
		}
	}
	return out
}

// FocusablePanels returns visible + focusable panels for FocusManager.
func (e *LayoutEngine) FocusablePanels() []Panel {
	var out []Panel
	for _, s := range e.VisibleSlots() {
		if s.Focusable {
			out = append(out, s.Panel)
		}
	}
	return out
}

// ComputeHeights distributes available height among visible panels.
func (e *LayoutEngine) ComputeHeights() map[string]int {
	visible := e.VisibleSlots()
	heights := make(map[string]int, len(visible))

	fixedTotal := 0
	flexTotal := 0
	for _, s := range visible {
		borderH := 0
		if s.Border != BorderNone {
			borderH = 2
		}
		if s.Weight == 0 {
			h := s.Panel.Height()
			if h == 0 {
				h = 1
			}
			heights[s.Panel.ID()] = h
			fixedTotal += h + borderH
		} else {
			flexTotal += s.Weight
		}
	}

	// Add newlines between panels.
	if len(visible) > 1 {
		fixedTotal += len(visible) - 1
	}

	remaining := e.height - fixedTotal
	if remaining < 0 {
		remaining = 0
	}

	for _, s := range visible {
		if s.Weight > 0 {
			h := remaining
			if flexTotal > 0 {
				h = remaining * s.Weight / flexTotal
			}
			if h < s.MinHeight {
				h = s.MinHeight
			}
			heights[s.Panel.ID()] = h
		}
	}

	return heights
}

// Render produces the full TUI view string.
func (e *LayoutEngine) Render() string {
	visible := e.VisibleSlots()
	if len(visible) == 0 {
		return ""
	}

	// Sync FocusManager with visible focusable panels.
	e.focus.SetPanels(e.FocusablePanels())

	innerWidth := e.width - 2
	if innerWidth < 10 {
		innerWidth = 10
	}
	heights := e.ComputeHeights()
	depths := FocusDepths(e.focus.Count(), e.focus.ActiveIndex())

	var sb strings.Builder
	focusIdx := 0
	for i, slot := range visible {
		if i > 0 {
			sb.WriteByte('\n')
		}

		// Resize flex panels.
		if slot.Weight > 0 {
			slot.Panel.Update(ResizeMsg{Width: innerWidth, Height: heights[slot.Panel.ID()]})
		}

		content := slot.Panel.View(innerWidth)

		switch slot.Border {
		case BorderFocusDepth:
			depth := 1 // default unfocused
			if slot.Focusable && focusIdx < len(depths) {
				depth = depths[focusIdx]
			}
			sb.WriteString(RenderWithDepth(content, depth, e.width))
		case BorderOnly:
			focused := false
			if slot.Focusable && focusIdx < len(depths) {
				focused = depths[focusIdx] == 0
			}
			sb.WriteString(RenderBorderOnly(content, focused, e.width))
		case BorderNone:
			sb.WriteString(content)
		}

		if slot.Focusable {
			focusIdx++
		}
	}

	return sb.String()
}
