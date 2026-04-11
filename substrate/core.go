// core.go — HubCore composition struct and DisplaySender contract (GOL-58).
//
// Every DevOps phase hub embeds HubCore for unified access to tracing,
// signals, and display. Five-step mediation: execute -> trace -> signal -> render -> sync.
package substrate

import (
	"github.com/dpopsuev/djinn/telemetry"
)

// HubCore is the shared infrastructure embedded by every concrete hub.
// Composition, not inheritance — each hub adds domain-specific fields.
// All methods are nil-safe.
type HubCore struct {
	Tracer  *telemetry.Tracer
	Signals *telemetry.SignalBus
	Display DisplaySender
}

// Trace records a point-in-time event through the tracer.
// Nil-safe: no-op when Tracer is nil.
func (c *HubCore) Trace(action, detail string) {
	if c.Tracer != nil {
		c.Tracer.Event(action, detail)
	}
}

// Emit emits a signal through the bus.
// Nil-safe: no-op when Signals is nil.
func (c *HubCore) Emit(s telemetry.Signal) {
	if c.Signals != nil {
		c.Signals.Emit(s)
	}
}

// Render sends a display message to the TUI.
// Nil-safe: no-op when Display is nil.
func (c *HubCore) Render(msg DisplayMsg) {
	if c.Display != nil {
		c.Display.Send(msg)
	}
}

// DisplaySender is the typed TUI contract for hub-to-display communication.
type DisplaySender interface {
	Send(msg DisplayMsg)
}

// DisplayMsg carries structured display data from a hub to the TUI.
type DisplayMsg struct {
	Source   string // hub name, e.g. "plan", "code", "test"
	Category string // DevOps phase
	Content  any    // hub-specific payload
}

// NopDisplaySender is a no-op implementation for tests and headless mode.
type NopDisplaySender struct{}

// Send is a no-op.
func (NopDisplaySender) Send(DisplayMsg) {}

// Compile-time interface check.
var _ DisplaySender = NopDisplaySender{}
