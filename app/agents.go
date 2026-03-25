// agents.go — bridges Bugle AgentPool signals to TUI agent messages.
// When an agent starts/stops/errors, the bridge emits AgentStatusMsg
// to the Bubbletea program so the AgentsPanel can render live status.
package app

import (
	"github.com/dpopsuev/djinn/bugleport"
	"github.com/dpopsuev/djinn/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// BridgeAgentPoolToTUI subscribes to Bugle Bus signals and forwards
// agent lifecycle events to the Bubbletea program as TUI messages.
func BridgeAgentPoolToTUI(bus bugleport.Bus, program *tea.Program) {
	bus.OnEmit(func(sig bugleport.Signal) {
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
	})
}
