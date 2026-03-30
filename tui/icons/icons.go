// icons.go — Nerd Font icon registry with ASCII fallback.
//
// Auto-detects Nerd Font support via DJINN_NERD_FONTS=1 env var.
// Falls back to ASCII glyphs when Nerd Fonts aren't available.
package icons

import "os"

// NerdFontsAvailable is true when Nerd Font glyphs can be rendered.
// Set via DJINN_NERD_FONTS=1 environment variable.
var NerdFontsAvailable = os.Getenv("DJINN_NERD_FONTS") == "1"

// Icon holds a Nerd Font glyph and its ASCII fallback.
type Icon struct {
	Nerd  string
	ASCII string
}

// String returns the appropriate glyph for the current terminal.
func (i Icon) String() string {
	if NerdFontsAvailable {
		return i.Nerd
	}
	return i.ASCII
}

// Semantic icon registry.
var (
	File    = Icon{"\uf15b", "F"}
	Folder  = Icon{"\uf07b", "D"}
	Git     = Icon{"\ue725", "G"}
	Branch  = Icon{"\ue725", "B"}
	Tag     = Icon{"\uf02b", "T"}
	Check   = Icon{"\uf00c", "✓"}
	Cross   = Icon{"\uf00d", "✗"}
	Warning = Icon{"\uf071", "!"}
	Info    = Icon{"\uf05a", "i"}
	Error   = Icon{"\uf06a", "E"}
	Spinner = Icon{"\uf110", "*"}
	Agent   = Icon{"\uf2bd", "A"}
	Tool    = Icon{"\uf0ad", "λ"}
	Clock   = Icon{"\uf017", "⏱"}
	Budget  = Icon{"\uf155", "$"}
)
