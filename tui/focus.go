// focus.go — depth-based brightness dimming for panel focus indication.
// Focused panel gets a rounded border. Unfocused panels dim progressively.
package tui

import "github.com/charmbracelet/lipgloss"

// Focus depth styles — progressively dimmer.
var (
	depthDim1 = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#808080", Dark: "#707070"})
	depthDim2 = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#a0a0a0", Dark: "#404040"})
)

// Focused panel border style — rounded edges, RedHatRed accent.
var focusBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(RedHatRed)

// RenderWithDepth wraps panel content with depth-based dimming.
// Depth 0 = focused (rounded border). Higher depth = dimmer, no border.
func RenderWithDepth(content string, depth int) string {
	switch {
	case depth <= 0:
		return focusBorder.Render(content)
	case depth == 1:
		return depthDim1.Render(content)
	default:
		return depthDim2.Render(content)
	}
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
