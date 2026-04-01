// focus_hint.go — Agent-to-TUI visual hints (GOL-62, SPC-85).
//
// SightHintMsg lets the agent guide operator attention by highlighting
// or scrolling to specific elements in panels.
package tui

// SightHintMsg is emitted by the agent to highlight an element in a panel.
// The target panel handles it in Update() — silently ignored if element not found.
type SightHintMsg struct {
	PanelID string `json:"panel"`
	CellID  string `json:"cell"`
	Action  string `json:"action"` // "highlight", "pulse", "scroll-to"
}
