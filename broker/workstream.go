package broker

import (
	"errors"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

// Sentinel errors for workstream operations.
var (
	ErrConcurrencyLimit = errors.New("concurrency limit reached")
)

// WorkstreamStatus represents the lifecycle state of a workstream.
type WorkstreamStatus string

const (
	WorkstreamRunning   WorkstreamStatus = "running"
	WorkstreamCompleted WorkstreamStatus = "completed"
	WorkstreamFailed    WorkstreamStatus = "failed"
	WorkstreamCancelled WorkstreamStatus = "cancelled"
	WorkstreamPending   WorkstreamStatus = "pending"
)

// WorkstreamInfo tracks metadata about a running or completed workstream.
type WorkstreamInfo struct {
	ID        string
	IntentID  string
	Action    string
	Status    WorkstreamStatus
	Scopes    []tier.Scope
	Health    signal.FlagLevel
	Bus       *signal.SignalBus // per-workstream signal partition
	StartedAt time.Time
	EndedAt   time.Time
}

// RegistryOption configures a WorkstreamRegistry.
type RegistryOption func(*WorkstreamRegistry)

// WithMaxConcurrent sets the maximum number of concurrent workstreams.
// Zero means unlimited.
func WithMaxConcurrent(n int) RegistryOption {
	return func(r *WorkstreamRegistry) { r.maxConcurrent = n }
}

// WorkstreamRegistry tracks active and recent workstreams.
type WorkstreamRegistry struct {
	mu             sync.RWMutex
	workstreams    map[string]*WorkstreamInfo
	maxConcurrent  int
	pending        []*WorkstreamInfo
}

// NewWorkstreamRegistry creates a new workstream registry.
func NewWorkstreamRegistry(opts ...RegistryOption) *WorkstreamRegistry {
	r := &WorkstreamRegistry{
		workstreams: make(map[string]*WorkstreamInfo),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// TryRegister attempts to register a workstream. Returns false and queue
// position if at the concurrency limit. The workstream is added to the
// pending queue with status WorkstreamPending.
func (r *WorkstreamRegistry) TryRegister(info *WorkstreamInfo) (bool, int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.maxConcurrent > 0 && r.activeCountLocked() >= r.maxConcurrent {
		info.Status = WorkstreamPending
		r.pending = append(r.pending, info)
		r.workstreams[info.ID] = info
		return false, len(r.pending)
	}

	info.Status = WorkstreamRunning
	r.workstreams[info.ID] = info
	return true, 0
}

// Register adds a workstream unconditionally (ignores concurrency limit).
func (r *WorkstreamRegistry) Register(info *WorkstreamInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workstreams[info.ID] = info
}

// Dequeue removes and returns the next pending workstream, promoting it to Running.
// Returns nil if no pending workstreams exist.
func (r *WorkstreamRegistry) Dequeue() *WorkstreamInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pending) == 0 {
		return nil
	}
	ws := r.pending[0]
	r.pending = r.pending[1:]
	ws.Status = WorkstreamRunning
	ws.StartedAt = time.Now()
	return ws
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

// ActiveCount returns the number of running workstreams.
func (r *WorkstreamRegistry) ActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeCountLocked()
}

func (r *WorkstreamRegistry) activeCountLocked() int {
	count := 0
	for _, ws := range r.workstreams {
		if ws.Status == WorkstreamRunning {
			count++
		}
	}
	return count
}

// PendingCount returns the number of queued workstreams.
func (r *WorkstreamRegistry) PendingCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.pending)
}

// All returns all tracked workstreams (running + completed + pending).
func (r *WorkstreamRegistry) All() []WorkstreamInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WorkstreamInfo, 0, len(r.workstreams))
	for _, ws := range r.workstreams {
		out = append(out, *ws)
	}
	return out
}

// FindByScope returns a running workstream whose scopes overlap with the given scopes.
func (r *WorkstreamRegistry) FindByScope(scopes []tier.Scope) (WorkstreamInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ws := range r.workstreams {
		if ws.Status != WorkstreamRunning {
			continue
		}
		if scopesOverlap(ws.Scopes, scopes) {
			return *ws, true
		}
	}
	return WorkstreamInfo{}, false
}

func scopesOverlap(a, b []tier.Scope) bool {
	for _, sa := range a {
		for _, sb := range b {
			if sa.Name == sb.Name {
				return true
			}
		}
	}
	return false
}
