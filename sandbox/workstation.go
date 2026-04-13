// workstation.go — Persistent agent-independent execution environment (TSK-651, TSK-653).
//
// A Workstation is bound to one artifact task and survives agent relay.
// Agents attach/detach — the workstation (tools, sandbox, scratch paper) persists.
// Attach fails if the workstation is occupied (must Detach first).
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// Sentinel errors for workstation operations.
var (
	ErrWorkstationOccupied = errors.New("workstation: occupied")
	ErrWorkstationVacant   = errors.New("workstation: vacant")
	ErrWorkstationNotFound = errors.New("workstation: not found")
)

// WorkstationID uniquely identifies a workstation.
type WorkstationID string

// Workstation is a persistent execution environment bound to one artifact task.
// Agents come and go; the workstation keeps tools, sandbox, and scratch paper.
type Workstation struct {
	ID        WorkstationID
	TaskID    string // bound to one artifact task
	ScratchID string // artifact ID of scratch paper child
	Sandbox   string // sandbox handle reference
	Agent     string // current tenant agent ID (empty = vacant)
	Created   time.Time
	mu        sync.RWMutex
}

// NewWorkstation creates a vacant workstation bound to a task.
func NewWorkstation(id WorkstationID, taskID string) *Workstation {
	return &Workstation{
		ID:      id,
		TaskID:  taskID,
		Created: time.Now(),
	}
}

// IsVacant returns true if no agent is currently attached.
func (w *Workstation) IsVacant() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Agent == ""
}

// Attach assigns an agent to this workstation.
// Fails with ErrWorkstationOccupied if another agent is already attached.
func (w *Workstation) Attach(agentID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Agent != "" {
		slog.WarnContext(context.Background(), "workstation attach rejected: not vacant",
			slog.String(telemetry.KeyWorkstreamID, string(w.ID)),
			slog.String(telemetry.KeyAgent, w.Agent),
			slog.String("rejected_agent", agentID),
		)
		return fmt.Errorf("%w: agent %q is attached", ErrWorkstationOccupied, w.Agent)
	}

	w.Agent = agentID
	slog.InfoContext(context.Background(), "workstation attach",
		slog.String(telemetry.KeyWorkstreamID, string(w.ID)),
		slog.String(telemetry.KeyAgent, agentID),
		slog.String(telemetry.KeyTaskID, w.TaskID),
	)
	return nil
}

// Detach removes the current agent from this workstation, returning the former
// agent ID and marking the workstation as vacant.
func (w *Workstation) Detach() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	former := w.Agent
	w.Agent = ""

	slog.InfoContext(context.Background(), "workstation detach",
		slog.String(telemetry.KeyWorkstreamID, string(w.ID)),
		"former_agent", former,
		slog.String(telemetry.KeyTaskID, w.TaskID),
	)
	return former
}
