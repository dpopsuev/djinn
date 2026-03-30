// focus_hint.go — Agent-to-TUI visual hints (GOL-62, SPC-85).
//
// FocusHintMsg lets the agent guide operator attention by highlighting
// or scrolling to specific elements in panels.
package tui

// FocusHintMsg is emitted by the agent to highlight an element in a panel.
// The target panel handles it in Update() — silently ignored if element not found.
type FocusHintMsg struct {
	PanelID   string `json:"panel"`
	ElementID string `json:"element"`
	Action    string `json:"action"` // "highlight", "pulse", "scroll-to"
}
