// elements.go — Re-exports from tui/elements/ for backward compatibility.
//
// Layer 1: atomic visual primitives. Actual implementations in tui/elements/.
package tui

import "github.com/dpopsuev/djinn/tui/elements"

// State constants — re-exported from tui/elements/.
const (
	StateDone    = elements.StateDone
	StateActive  = elements.StateActive
	StateError   = elements.StateError
	StatePending = elements.StatePending
)

// Glyph returns a styled status glyph for the given state.
func Glyph(state string) string { return elements.Glyph(state) }

// Badge renders a labeled value: Badge("tokens", 8150) → "8.2k tokens".
func Badge(label string, value int) string { return elements.Badge(label, value) }

// Hint renders keybinding hints separated by middle dots.
func Hint(bindings ...string) string { return elements.Hint(bindings...) }

// HorizontalRule renders a horizontal line of the given width.
func HorizontalRule(width int) string { return elements.HorizontalRule(width) }

// Dim applies faint styling to content for visual hierarchy.
func Dim(content string) string { return elements.Dim(content) }

// CompactNumber formats large numbers: 1200→"1.2k", 3400000→"3.4M", 42→"42".
func CompactNumber(n int) string { return elements.CompactNumber(n) }
