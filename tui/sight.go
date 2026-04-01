// sight.go — Bidirectional TUI state in agent prompt (GOL-62, SPC-85).
//
// Sighted is an optional interface panels implement to report
// what the operator is looking at. On SubmitMsg, the active panel's cell
// sight is injected into the agent prompt so "this" resolves from TUI state.
package tui

import (
	"fmt"
	"strings"
)

// CellSight describes what the operator is currently focused on in a panel.
type CellSight struct {
	PanelID   string            `json:"panel"`
	CellID    string            `json:"cell,omitempty"`
	CellTitle string            `json:"title,omitempty"`
	Kind      string            `json:"kind,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// IsEmpty returns true if the context carries no meaningful focus information.
func (fc CellSight) IsEmpty() bool {
	return fc.PanelID == "" && fc.CellID == ""
}

// FormatPrompt renders the cell sight as a structured prompt block.
// Output is under 200 tokens — context, not content.
func (fc CellSight) FormatPrompt() string {
	if fc.IsEmpty() {
		return ""
	}

	var b strings.Builder
	b.WriteString("<focus-context>\n")
	fmt.Fprintf(&b, "  Panel: %s\n", fc.PanelID)
	if fc.CellID != "" {
		fmt.Fprintf(&b, "  Selected: %s", fc.CellID)
		if fc.CellTitle != "" {
			fmt.Fprintf(&b, " %q", fc.CellTitle)
		}
		b.WriteByte('\n')
	}
	if fc.Kind != "" {
		fmt.Fprintf(&b, "  Kind: %s\n", fc.Kind)
	}
	for k, v := range fc.Metadata {
		fmt.Fprintf(&b, "  %s: %s\n", k, v)
	}
	b.WriteString("</focus-context>")
	return b.String()
}

// Sighted is an optional interface for panels that can report
// what the operator is focused on. Panels without selectable elements
// (e.g., InputPanel) don't implement this — backward compatible via type assertion.
type Sighted interface {
	CellSight() CellSight
}
