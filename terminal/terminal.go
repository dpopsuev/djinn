// Package terminal defines the programmatic API for driving Djinn.
//
// Three interfaces following ISP (Interface Segregation Principle):
//   - Controller: operator inputs (Submit, Shell, Command, configuration)
//   - Viewer: observable outputs (Subscribe, Status, Introspect)
//   - Terminal: Controller + Viewer + Lifecycle (the full facade)
//
// Consumers accept the narrowest interface they need:
//   - TUI accepts Terminal (reads + writes + lifecycle)
//   - TestOperator accepts Controller (writes only)
//   - Referee accepts Viewer (reads only)
//
// One concrete implementation backs all three interfaces.
package terminal

import "context"

// Controller handles operator inputs — what the operator does TO Djinn.
// TestOperator and MockOperator accept this interface.
type Controller interface {
	// Submit sends a natural language prompt to the active agent.
	Submit(ctx context.Context, prompt string) error

	// Shell executes a command on the host shell (! prefix).
	Shell(ctx context.Context, command string) (string, error)

	// Command executes an internal Djinn command (: prefix).
	Command(ctx context.Context, name string, args []string) (string, error)

	// SetOperation switches the active operation (ask/plan/agent).
	SetOperation(op string)

	// SetCapacity sets the maximum concurrent agent count.
	SetCapacity(n int) error

	// NavigateScope moves to a scope position with the given type.
	NavigateScope(path string, scopeType string) error

	// SetEnvelopeEnabled toggles the lifecycle envelope on or off.
	SetEnvelopeEnabled(enabled bool)
}

// Viewer handles observable outputs — what Djinn produces for observers.
// Referee and monitoring tools accept this interface.
type Viewer interface {
	// Subscribe registers a channel to receive ViewEvents.
	// Multiple subscribers are supported. Events are non-blocking sends.
	Subscribe(ch chan<- ViewEvent)

	// Unsubscribe removes a previously registered channel.
	Unsubscribe(ch chan<- ViewEvent)

	// Status returns the current run state snapshot.
	Status() RunState

	// Introspect returns a detailed report of what's enabled and active.
	Introspect() IntrospectionReport
}

// Terminal composes Controller + Viewer with lifecycle management.
// TUI and headless runners accept this interface.
type Terminal interface {
	Controller
	Viewer

	// Start initializes the terminal and begins accepting inputs.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the terminal.
	Stop()
}
