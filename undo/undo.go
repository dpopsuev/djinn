// Package undo provides checkpoint/rollback for agent workspaces.
//
// PoC: linear checkpoints. Checkpoint = EventLog index.
// Rollback = Mirage.Reset() + report undone actions.
// Inspired by Neovim's undo tree (NED-46).
package undo

import (
	"time"

	"github.com/dpopsuev/troupe/signal"
)

// Checkpoint captures a point-in-time workspace state.
type Checkpoint struct {
	Index     int       `json:"index"`     // EventLog index at checkpoint time
	Name      string    `json:"name"`      // human-readable label
	Timestamp time.Time `json:"timestamp"` // when the checkpoint was created
}

// Manager provides checkpoint/rollback over an EventLog + workspace.
type Manager interface {
	// Checkpoint saves the current EventLog position. Returns the index.
	Checkpoint(name string) int

	// Rollback restores the workspace to the given checkpoint index.
	// Returns the events that were undone (everything since the checkpoint).
	Rollback(index int) ([]signal.Event, error)

	// Current returns the most recent checkpoint index, or -1 if none.
	Current() int

	// Checkpoints returns all checkpoints in creation order.
	Checkpoints() []Checkpoint
}
