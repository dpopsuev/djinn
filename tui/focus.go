// focus.go — depth-based brightness dimming for panel focus indication.
// Focused panel renders at full brightness. Unfocused panels dim
// progressively by distance from focus in the panel tree.
package tui

import "github.com/charmbracelet/lipgloss"

// Focus depth styles — progressively dimmer.
var (
	// Depth 0: focused panel — full brightness (no transformation)
	// Depth 1: sibling — visibly muted (gray foreground)
	depthDim1 = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#808080", Dark: "#707070"})
	// Depth 2+: distant — very faint (dark gray)
	depthDim2 = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#a0a0a0", Dark: "#404040"})
)

// Focused panel separator accent
var focusAccent = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"}).
	Bold(true)

// RenderWithDepth wraps panel content with depth-based dimming.
// Depth 0 = focused (full brightness). Higher depth = dimmer.
func RenderWithDepth(content string, depth int) string {
	switch {
	case depth <= 0:
		return content // focused: full brightness
	case depth == 1:
		return depthDim1.Render(content) // sibling: visibly muted
	default:
		return depthDim2.Render(content) // distant: very faint
	}
}

// RenderFocusIndicator returns a focus marker for the active panel.
func RenderFocusIndicator(focused bool) string {
	if focused {
		return focusAccent.Render("▌")
	}
	return " "
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
