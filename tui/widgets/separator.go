// separator.go — panel separators with nesting-aware line weight.
package widgets

import (
	"strings"

	"github.com/dpopsuev/djinn/tui/design"

	tui "github.com/dpopsuev/djinn/tui"
)

// Separator characters by nesting depth.
const (
	SepThick  = '━' // level 0: root panel boundaries
	SepThin   = '─' // level 1: panel sections
	SepDotted = '┄' // level 2: child items
	SepSparse = '·' // level 3+: leaf details
)

// Separator renders a horizontal line at the given nesting depth.
// Focused separators render in accent color.
func Separator(width, depth int, focused bool) string {
	if width <= 0 {
		return ""
	}

	var ch rune
	switch {
	case depth <= 0:
		ch = SepThick
	case depth == 1:
		ch = SepThin
	case depth == 2:
		ch = SepDotted
	default:
		ch = SepSparse
	}

	var line string
	if ch == SepSparse {
		// Sparse: "· " repeated
		var sb strings.Builder
		for i := 0; i < width; i += 2 {
			sb.WriteRune(ch)
			if i+1 < width {
				sb.WriteByte(' ')
			}
		}
		line = sb.String()
	} else {
		line = strings.Repeat(string(ch), width)
	}

	if focused {
		return design.ActiveStyles.SepFocus.Render(line)
	}
	return tui.DimStyle.Render(line)
}
