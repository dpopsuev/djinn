// Package vezir defines the Control Plane daemon interface.
// Supervisor (Erlang OTP-style), reconciliation loop, socket relay,
// builder/watcher. Stateless — reads desired state from config,
// observes actual state from Substrate health, acts on diff.
package vezir

import "context"

// Vezir is the Control Plane. Supervises Substrate and TUI processes.
type Vezir interface {
	// Start begins supervision. Blocks until context is canceled.
	Start(ctx context.Context) error

	// Health returns the supervised processes' health.
	Health() HealthReport

	// Restart triggers a restart of the named process.
	Restart(ctx context.Context, process string) error
}

// HealthReport summarizes the state of supervised processes.
type HealthReport struct {
	Substrate ProcessState `json:"substrate"`
	TUI       ProcessState `json:"tui"`
}

// ProcessState describes a supervised process.
type ProcessState struct {
	Running  bool   `json:"running"`
	PID      int    `json:"pid,omitempty"`
	Restarts int    `json:"restarts"`
	LastErr  string `json:"last_error,omitempty"`
}
