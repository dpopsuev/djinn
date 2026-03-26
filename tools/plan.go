// plan.go — in-process task tracker for agent planning.
//
// TaskStore is a thread-safe, file-backed task store with dependency ordering.
// Agents use it to decompose work into tasks, track progress, and
// topologically sort by dependency edges.
package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Task represents a single work item in the plan.
type Task struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"` // pending, active, done, blocked
	DependsOn []string `json:"depends_on,omitempty"`
	Parent    string   `json:"parent,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

// Valid task statuses.
const (
	StatusPending = "pending"
	StatusActive  = "active"
	StatusDone    = "done"
	StatusBlocked = "blocked"
)

// TaskStore is a thread-safe, file-backed collection of tasks.
type TaskStore struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	path   string
	nextID int
}

// NewTaskStore creates a TaskStore backed by the given file path.
// If the file exists, it is loaded on first access via Load().
func NewTaskStore(path string) *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
		path:  path,
	}
}

// Create adds a new task with the given title and returns it.
func (s *TaskStore) Create(title string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	now := time.Now()
	t := &Task{
		ID:      fmt.Sprintf("T-%03d", s.nextID),
		Title:   title,
		Status:  StatusPending,
		Created: now,
		Updated: now,
	}
	s.tasks[t.ID] = t
	return t
}

// Get returns a task by ID.
func (s *TaskStore) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	return t, ok
}

// Update changes a task's status. Returns an error if the task
// is not found or the status is invalid.
func (s *TaskStore) Update(id string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	switch status {
	case StatusPending, StatusActive, StatusDone, StatusBlocked:
	default:
		return fmt.Errorf("invalid status %q", status)
	}
	t.Status = status
	t.Updated = time.Now()
	return nil
}

// List returns all tasks sorted by ID.
func (s *TaskStore) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TopoSort returns tasks in dependency order (dependencies before dependents).
// Tasks with no dependencies come first. Cycles are broken arbitrarily.
func (s *TaskStore) TopoSort() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Kahn's algorithm.
	inDegree := make(map[string]int, len(s.tasks))
	dependents := make(map[string][]string) // dep → tasks that depend on it

	for id, t := range s.tasks {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, dep := range t.DependsOn {
			if _, ok := s.tasks[dep]; ok {
				inDegree[id]++
				dependents[dep] = append(dependents[dep], id)
			}
		}
	}

	// Seed queue with zero-indegree nodes.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue) // deterministic order

	var sorted []*Task
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, s.tasks[id])
		for _, child := range dependents[id] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
		sort.Strings(queue) // keep deterministic
	}

	// If there are leftover tasks (cycles), append them.
	if len(sorted) < len(s.tasks) {
		remaining := make(map[string]bool)
		for id := range s.tasks {
			remaining[id] = true
		}
		for _, t := range sorted {
			delete(remaining, t.ID)
		}
		var ids []string
		for id := range remaining {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			sorted = append(sorted, s.tasks[id])
		}
	}

	return sorted
}

// Save writes the task store to disk as JSON. The write is atomic
// (write to temp then rename).
func (s *TaskStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.marshalState(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// Load reads task data from the file. Existing in-memory tasks are replaced.
func (s *TaskStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read tasks: %w", err)
	}

	var state taskStoreState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal tasks: %w", err)
	}

	s.tasks = make(map[string]*Task, len(state.Tasks))
	s.nextID = state.NextID
	for i := range state.Tasks {
		t := state.Tasks[i]
		s.tasks[t.ID] = &t
	}
	return nil
}

// taskStoreState is the JSON-serialized form.
type taskStoreState struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}

func (s *TaskStore) marshalState() taskStoreState {
	tasks := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, *t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return taskStoreState{NextID: s.nextID, Tasks: tasks}
}
