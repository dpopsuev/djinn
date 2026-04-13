// workstation_mgr.go — WorkstationManager: lifecycle for workstations (TSK-654).
//
// Creates, tracks, assigns, and releases workstations. Thread-safe.
// Auto-generates WorkstationIDs from an incrementing counter.
package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/dpopsuev/djinn/telemetry"
)

// WorkstationManager manages the lifecycle of workstations.
type WorkstationManager struct {
	workstations map[WorkstationID]*Workstation
	mu           sync.RWMutex
	log          *slog.Logger
	counter      atomic.Int64
}

// NewWorkstationManager creates a manager ready to track workstations.
func NewWorkstationManager(log *slog.Logger) *WorkstationManager {
	if log == nil {
		log = telemetry.Nop()
	}
	return &WorkstationManager{
		workstations: make(map[WorkstationID]*Workstation),
		log:          log,
	}
}

// Create provisions a new workstation bound to a task and returns it.
func (m *WorkstationManager) Create(taskID string) *Workstation {
	id := WorkstationID(fmt.Sprintf("WS-%03d", m.counter.Add(1)))

	ws := NewWorkstation(id, taskID)

	m.mu.Lock()
	m.workstations[id] = ws
	m.mu.Unlock()

	m.log.InfoContext(context.Background(), "workstation created",
		"workstation", id,
		"task", taskID,
	)
	return ws
}

// Get returns a workstation by ID, or false if not found.
func (m *WorkstationManager) Get(id WorkstationID) (*Workstation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ws, ok := m.workstations[id]
	return ws, ok
}

// Release removes a workstation from management. The workstation should be
// vacant before release (the caller is responsible for detaching first).
func (m *WorkstationManager) Release(id WorkstationID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.workstations[id]; ok {
		delete(m.workstations, id)
		m.log.InfoContext(context.Background(), "workstation released", slog.String(telemetry.KeyWorkstreamID, string(id)))
	}
}

// List returns all managed workstations as a snapshot.
func (m *WorkstationManager) List() []*Workstation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Workstation, 0, len(m.workstations))
	for _, ws := range m.workstations {
		out = append(out, ws)
	}
	return out
}

// Assign attaches an agent to a workstation by ID. Delegates to Workstation.Attach.
func (m *WorkstationManager) Assign(id WorkstationID, agentID string) error {
	m.mu.RLock()
	ws, ok := m.workstations[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("workstation %s: %w", id, ErrWorkstationNotFound)
	}
	return ws.Attach(agentID)
}
