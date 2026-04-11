// display_adapter.go — TeaDisplayAdapter bridges hubs to Bubbletea (GOL-58).
//
// Sends DisplayMsg as tea.Msg to the program's message loop.
// Nil-safe: no-op when program is nil.
package substrate

import tea "github.com/charmbracelet/bubbletea"

// TeaDisplayAdapter bridges DisplaySender to a Bubbletea program.
type TeaDisplayAdapter struct {
	program *tea.Program
}

// NewTeaDisplayAdapter creates a display adapter for the given program.
func NewTeaDisplayAdapter(p *tea.Program) *TeaDisplayAdapter {
	return &TeaDisplayAdapter{program: p}
}

// Send delivers a DisplayMsg to the Bubbletea message loop.
// Nil-safe: no-op when adapter or program is nil.
func (a *TeaDisplayAdapter) Send(msg DisplayMsg) {
	if a == nil || a.program == nil {
		return
	}
	a.program.Send(msg)
}

// Compile-time interface check.
var _ DisplaySender = (*TeaDisplayAdapter)(nil)
