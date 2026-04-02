// andon.go — Andon visual system types for the Djinn TUI.
//
// Andon is a manufacturing term for a visual indicator of health.
// AndonLevel maps to dashboard glyphs: green (healthy), yellow (warning), red (critical).
// AndonUpdateMsg bridges the broker's AndonBoard to the TUI dashboard.
package tui

// AndonLevel represents the visual health indicator level.
type AndonLevel int

const (
	AndonGreen  AndonLevel = iota // healthy — steady glyph
	AndonYellow                   // warning — slow pulse
	AndonRed                      // critical — fast blink
)

// String returns the level name.
func (l AndonLevel) String() string {
	switch l {
	case AndonGreen:
		return "green"
	case AndonYellow:
		return "yellow"
	case AndonRed:
		return "red"
	default:
		return "unknown"
	}
}

// AndonState represents the current andon indicator state.
type AndonState struct {
	Level   AndonLevel
	Source  string // which component triggered (e.g., "budget", "gate")
	Message string
}

// AndonUpdateMsg carries an andon state change to the TUI.
type AndonUpdateMsg struct {
	State AndonState
}
