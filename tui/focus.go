// focus.go — depth-based panel focus indication via border color.
// All panels always have borders (no height jump on focus change).
// Focused = RedHatRed border. Unfocused = dim grey border.
package tui

import "github.com/charmbracelet/lipgloss"

// Focused panel border — rounded edges, RedHatRed accent.
var focusBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(RedHatRed)

// Unfocused panel border — same shape, dim grey.
var unfocusedBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#808080", Dark: "#505050"})

// RenderWithDepth wraps panel content with a border at the given width.
// Depth 0 = focused (RedHatRed border). Higher depth = unfocused (dim grey border).
// Width is the OUTER width (including border chars).
func RenderWithDepth(content string, depth, width int) string {
	// Width sets inner content width (lipgloss subtracts border from Width).
	if depth <= 0 {
		return focusBorder.Width(width - 2).Render(content)
	}
	return unfocusedBorder.Width(width - 2).Render(content)
}

// RenderBorderOnly applies a border without foreground dimming.
// Use for panels that must never have their content dimmed (e.g., output panel
// where dimming causes ANSI escape codes to show as visible text — DJN-BUG-11).
func RenderBorderOnly(content string, focused bool, width int) string {
	if focused {
		return focusBorder.Width(width - 2).Render(content)
	}
	return unfocusedBorder.Width(width - 2).Render(content)
}

// RenderFocusIndicator returns a focus marker for the active panel.
func RenderFocusIndicator(focused bool) string {
	if focused {
		return ""
	}
	return ""
}

// FocusDepths calculates the focus depth for each panel in a flat list.
// The focused panel (by index) gets depth 0. Distance from focus = depth.
func FocusDepths(count, focusedIdx int) []int {
	depths := make([]int, count)
	for i := range depths {
		d := i - focusedIdx
		if d < 0 {
			d = -d
		}
		depths[i] = d
	}
	return depths
}
