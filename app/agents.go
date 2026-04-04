// agents.go — bridges Bugle Staff signals to TUI agent messages.
// When an agent starts/stops/errors, the bridge emits AgentStatusMsg
// to the Bubbletea program so the AgentsPanel can render live status.
package app

import (
	"github.com/dpopsuev/djinn/jerichoport"
	"github.com/dpopsuev/djinn/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// bridgeSignalHandler returns a signal handler that forwards agent lifecycle
// events to the Bubbletea program as TUI messages.
func bridgeSignalHandler(program *tea.Program) func(jerichoport.Signal) {
	return func(sig jerichoport.Signal) {
		switch sig.Event {
		case jerichoport.EventWorkerStarted:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[jerichoport.MetaKeyWorkerID],
				Role:    sig.Meta["role"],
				State:   "idle",
				Color:   sig.Meta["color"], // hex from Display, empty = TUI uses default
			})
		case jerichoport.EventWorkerStopped:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[jerichoport.MetaKeyWorkerID],
				State:   "done",
			})
		case jerichoport.EventWorkerError:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[jerichoport.MetaKeyWorkerID],
				State:   "error",
			})
		case jerichoport.EventWorkerDone:
			program.Send(tui.AgentStatusMsg{
				AgentID: sig.Meta[jerichoport.MetaKeyWorkerID],
				State:   "done",
			})
		}
	}
}

// BridgeStaffToTUI subscribes to a Staff's signal bus and forwards
// agent lifecycle events to the Bubbletea program as TUI messages.
// Uses the facade — no raw signal.Meta parsing needed.
func BridgeStaffToTUI(staff *jerichoport.Staff, program *tea.Program) {
	staff.OnSignal(bridgeSignalHandler(program))
}
