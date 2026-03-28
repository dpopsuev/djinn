// agents.go — bridges Bugle Staff signals to TUI agent messages.
// When an agent starts/stops/errors, the bridge emits AgentStatusMsg
// to the Bubbletea program so the AgentsPanel can render live status.
package app

import (
	"github.com/dpopsuev/djinn/bugleport"
	"github.com/dpopsuev/djinn/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// bridgeSignalHandler returns a signal handler that forwards agent lifecycle
// events to the Bubbletea program as TUI messages.
func bridgeSignalHandler(program *tea.Program) func(bugleport.Signal) {
	return func(sig bugleport.Signal) {
		switch sig.Event {
		case bugleport.EventWorkerStarted:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[bugleport.MetaKeyWorkerID],
				Role:    sig.Meta["role"],
				State:   "idle",
			})
		case bugleport.EventWorkerStopped:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[bugleport.MetaKeyWorkerID],
				State:   "done",
			})
		case bugleport.EventWorkerError:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[bugleport.MetaKeyWorkerID],
				State:   "error",
			})
		case bugleport.EventWorkerDone:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[bugleport.MetaKeyWorkerID],
				State:   "done",
			})
		}
	}
}

// BridgeStaffToTUI subscribes to a Staff's signal bus and forwards
// agent lifecycle events to the Bubbletea program as TUI messages.
// Uses the facade — no raw signal.Meta parsing needed.
func BridgeStaffToTUI(staff *bugleport.Staff, program *tea.Program) {
	staff.OnSignal(bridgeSignalHandler(program))
}

// BridgeAgentPoolToTUI is the legacy bridge using raw Bus.
// Deprecated: use BridgeStaffToTUI with the facade instead.
func BridgeAgentPoolToTUI(bus bugleport.Bus, program *tea.Program) {
	bus.OnEmit(bridgeSignalHandler(program))
}
