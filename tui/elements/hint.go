// hint.go — Keybinding hints and horizontal rules.
//
// Reads from design.ActiveStyles for dim styling.
package elements

import (
	"strings"

	"github.com/dpopsuev/djinn/tui/design"
)

// Hint renders keybinding hints separated by middle dots.
// Hint("enter send", "↑ edit", "esc cancel") → "enter send · ↑ edit · esc cancel"
func Hint(bindings ...string) string {
	if len(bindings) == 0 {
		return ""
	}
	return design.ActiveStyles.Dim.Render(strings.Join(bindings, " · "))
}

// HorizontalRule renders a horizontal line of the given width.
func HorizontalRule(width int) string {
	if width <= 0 {
		return ""
	}
	return design.ActiveStyles.Dim.Render(strings.Repeat("─", width))
}

// Dim applies faint styling to content for visual hierarchy.
func Dim(content string) string {
	return design.ActiveStyles.Dim.Render(content)
}
