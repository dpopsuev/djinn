// plan.go — Backward-compatibility shim: TaskStore wraps artifact.Graph (GOL-59).
//
// Task is an alias for artifact.Artifact. TaskStore delegates to artifact.Graph
// with Kind=task. All 9 tools/plan_test.go tests pass unchanged.
// Persistence uses the original JSON format for backward compatibility.
package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/dpopsuev/djinn/artifact"
)

// Task is an alias for artifact.Artifact.
type Task = artifact.Artifact

// Status type alias for backward compatibility.
type Status = artifact.Status

// Sentinel errors re-exported.
var (
	ErrTaskNotFound      = artifact.ErrNotFound
	ErrInvalidTaskStatus = artifact.ErrInvalidStatus
)

// Valid task statuses — re-exported from artifact.
const (
	StatusPending = artifact.StatusPending
	StatusActive  = artifact.StatusActive
	StatusDone    = artifact.StatusDone
	StatusBlocked = artifact.StatusBlocked
)

// TaskStore is a thread-safe, file-backed collection of tasks.
// Wraps artifact.Graph with Kind=task.
type TaskStore struct {
	g    *artifact.Graph
	path string
}

// NewTaskStore creates a TaskStore backed by the given file path.
func NewTaskStore(path string) *TaskStore {
	return &TaskStore{
		g:    artifact.NewGraph("tasks", artifact.DefaultRegistry()),
		path: path,
	}
}

// Create adds a new task with the given title and returns it.
func (s *TaskStore) Create(title string) *Task {
	a := artifact.Artifact{
		Kind:   artifact.KindTask,
		Title:  title,
		Status: artifact.StatusPending,
	}
	id, _ := s.g.Add(a) //nolint:errcheck // original API never returned error
	t, _ := s.g.Get(id)
	return t
}

// Get returns a task by ID.
func (s *TaskStore) Get(id string) (*Task, bool) {
	t, err := s.g.Get(id)
	return t, err == nil
}

// Update changes a task's status.
func (s *TaskStore) Update(id string, status Status) error {
	return s.g.UpdateStatus(id, status)
}

// List returns all tasks sorted by ID.
func (s *TaskStore) List() []*Task {
	all := s.g.ListSorted()
	out := make([]*Task, len(all))
	for i := range all {
		t, _ := s.g.Get(all[i].ID)
		out[i] = t
	}
	return out
}

// TopoSort returns tasks in dependency order.
func (s *TaskStore) TopoSort() []*Task {
	sorted := s.g.TopoSort()
	out := make([]*Task, len(sorted))
	for i := range sorted {
		t, _ := s.g.Get(sorted[i].ID)
		out[i] = t
	}
	return out
}

// Save writes the task store to disk as JSON (original format for backward compat).
func (s *TaskStore) Save() error {
	all := s.g.ListSorted()
	tasks := make([]taskJSON, len(all))
	for i := range all {
		tasks[i] = toTaskJSON(&all[i])
	}
	state := taskStoreState{
		NextID: int(s.g.CounterValue(artifact.KindTask)),
		Tasks:  tasks,
	}
	return atomicSaveJSON(s.path, state, "tasks")
}

// Load reads task data from the file (original format).
func (s *TaskStore) Load() error {
	data, err := os.ReadFile(s.path) //nolint:gosec // path from controlled config
	if err != nil {
		return fmt.Errorf("read tasks: %w", err)
	}

	var state taskStoreState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal tasks: %w", err)
	}

	// Reset graph and restore.
	s.g = artifact.NewGraph("tasks", artifact.DefaultRegistry())
	for i := range state.Tasks {
		t := state.Tasks[i]
		a := artifact.Artifact{
			ID:        t.ID,
			Kind:      artifact.KindTask,
			Title:     t.Title,
			Status:    artifact.Status(t.Status),
			DependsOn: t.DependsOn,
			Parent:    t.Parent,
			Labels:    t.Labels,
			Created:   t.Created,
			Updated:   t.Updated,
		}
		s.g.Add(a) //nolint:errcheck // loading known-good data
	}
	s.g.SetCounter(artifact.KindTask, int64(state.NextID))
	return nil
}

// taskJSON is the original JSON format for a task (backward compat).
type taskJSON struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	DependsOn []string  `json:"depends_on,omitempty"`
	Parent    string    `json:"parent,omitempty"`
	Labels    []string  `json:"labels,omitempty"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

func toTaskJSON(a *artifact.Artifact) taskJSON {
	return taskJSON{
		ID:        a.ID,
		Title:     a.Title,
		Status:    string(a.Status),
		DependsOn: a.DependsOn,
		Parent:    a.Parent,
		Labels:    a.Labels,
		Created:   a.Created,
		Updated:   a.Updated,
	}
}

// taskStoreState is the original JSON format.
type taskStoreState struct {
	NextID int        `json:"next_id"`
	Tasks  []taskJSON `json:"tasks"`
}

// --- Legacy sort helper (used by reconcile.go) ---

// SortTasksByID sorts tasks by ID. Used by tests and reconcile.
func SortTasksByID(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
}
