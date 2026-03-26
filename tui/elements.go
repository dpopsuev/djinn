// elements.go — Layer 1: atomic visual primitives.
//
// Pure functions, zero state. The smallest units of the TUI design system.
// Components (Layer 2) compose these. Panels (Layer 3) compose components.
// Dependency: Element → lipgloss styles. No upward references.
package tui

import (
	"fmt"
	"strings"
)

// Glyph returns a styled status glyph for the given state.
func Glyph(state string) string {
	switch state {
	case "done":
		return ToolSuccessStyle.Render("⬢")
	case "active":
		return ToolNameStyle.Render("⬡")
	case "error":
		return ErrorStyle.Render("●")
	case "pending":
		return DimStyle.Render("○")
	default:
		return DimStyle.Render("○")
	}
}

// Badge renders a labeled value: Badge("tokens", 8150) → "8.2k tokens".
func Badge(label string, value int) string {
	return CompactNumber(value) + " " + label
}

// Hint renders keybinding hints separated by middle dots.
// Hint("enter send", "↑ edit", "esc cancel") → "enter send · ↑ edit · esc cancel"
func Hint(bindings ...string) string {
	if len(bindings) == 0 {
		return ""
	}
	return DimStyle.Render(strings.Join(bindings, " · "))
}

// HorizontalRule renders a horizontal line of the given width.
func HorizontalRule(width int) string {
	if width <= 0 {
		return ""
	}
	return DimStyle.Render(strings.Repeat("─", width))
}

// CompactNumber formats large numbers: 1200→"1.2k", 3400000→"3.4M", 42→"42".
func CompactNumber(n int) string {
	if n < 0 {
		return "-" + CompactNumber(-n)
	}
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
