// glyph.go — Status glyph rendering.
//
// Reads from design.ActiveStyles to avoid importing tui/ (circular dep).
package elements

import "github.com/dpopsuev/djinn/tui/design"

// Glyph returns a styled status glyph for the given state.
func Glyph(state string) string {
	switch state {
	case StateDone:
		return design.ActiveStyles.ToolSuccess.Render("⬢")
	case StateActive:
		return design.ActiveStyles.ToolName.Render("⬡")
	case StateError:
		return design.ActiveStyles.Error.Render("●")
	case StatePending:
		return design.ActiveStyles.Dim.Render("○")
	default:
		return design.ActiveStyles.Dim.Render("○")
	}
}
