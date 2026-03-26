// icons.go — Nerd Font icon registry with ASCII fallback.
//
// Auto-detects Nerd Font support via DJINN_NERD_FONTS=1 env var.
// Falls back to ASCII glyphs when Nerd Fonts aren't available.
package tui

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
	IconFile    = Icon{"\uf15b", "F"}
	IconFolder  = Icon{"\uf07b", "D"}
	IconGit     = Icon{"\ue725", "G"}
	IconBranch  = Icon{"\ue725", "B"}
	IconTag     = Icon{"\uf02b", "T"}
	IconCheck   = Icon{"\uf00c", "✓"}
	IconCross   = Icon{"\uf00d", "✗"}
	IconWarning = Icon{"\uf071", "!"}
	IconInfo    = Icon{"\uf05a", "i"}
	IconError   = Icon{"\uf06a", "E"}
	IconSpinner = Icon{"\uf110", "*"}
	IconAgent   = Icon{"\uf2bd", "A"}
	IconTool    = Icon{"\uf0ad", "λ"}
	IconClock   = Icon{"\uf017", "⏱"}
	IconBudget  = Icon{"\uf155", "$"}
)
