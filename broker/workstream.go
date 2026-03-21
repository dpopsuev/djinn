package broker

import (
	"sync"
	"time"

	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

// WorkstreamStatus represents the lifecycle state of a workstream.
type WorkstreamStatus string

const (
	WorkstreamRunning   WorkstreamStatus = "running"
	WorkstreamCompleted WorkstreamStatus = "completed"
	WorkstreamFailed    WorkstreamStatus = "failed"
	WorkstreamCancelled WorkstreamStatus = "cancelled"
)

// WorkstreamInfo tracks metadata about a running or completed workstream.
type WorkstreamInfo struct {
	ID        string
	IntentID  string
	Action    string
	Status    WorkstreamStatus
	Scopes    []tier.Scope
	Health    signal.FlagLevel
	StartedAt time.Time
	EndedAt   time.Time
}

// WorkstreamRegistry tracks active and recent workstreams.
type WorkstreamRegistry struct {
	mu          sync.RWMutex
	workstreams map[string]*WorkstreamInfo
}

// NewWorkstreamRegistry creates a new workstream registry.
func NewWorkstreamRegistry() *WorkstreamRegistry {
	return &WorkstreamRegistry{
		workstreams: make(map[string]*WorkstreamInfo),
	}
}

// Register adds a new workstream.
func (r *WorkstreamRegistry) Register(info *WorkstreamInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workstreams[info.ID] = info
}

// Complete marks a workstream as completed, failed, or cancelled.
// Only transitions from Running are allowed — a cancelled workstream
// cannot be overwritten by a subsequent failed/completed status.
func (r *WorkstreamRegistry) Complete(id string, status WorkstreamStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ws, ok := r.workstreams[id]; ok && ws.Status == WorkstreamRunning {
		ws.Status = status
		ws.EndedAt = time.Now()
	}
}

// Get returns a workstream by ID.
func (r *WorkstreamRegistry) Get(id string) (WorkstreamInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ws, ok := r.workstreams[id]
	if !ok {
		return WorkstreamInfo{}, false
	}
	return *ws, true
}

// Active returns all running workstreams.
func (r *WorkstreamRegistry) Active() []WorkstreamInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []WorkstreamInfo
	for _, ws := range r.workstreams {
		if ws.Status == WorkstreamRunning {
			out = append(out, *ws)
		}
	}
	return out
}

// All returns all tracked workstreams (running + completed).
func (r *WorkstreamRegistry) All() []WorkstreamInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WorkstreamInfo, 0, len(r.workstreams))
	for _, ws := range r.workstreams {
		out = append(out, *ws)
	}
	return out
}
