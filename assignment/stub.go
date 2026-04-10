package assignment

import "sync"

var _ Manager = (*StubManager)(nil)

// StubManager implements Manager for testing.
type StubManager struct {
	mu          sync.Mutex
	assignments map[string]*Assignment
	nextID      int
}

func NewStubManager() *StubManager {
	return &StubManager{assignments: make(map[string]*Assignment)}
}

func (m *StubManager) Assign(agentID, task string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := agentID + "-" + string(rune('0'+m.nextID))
	m.assignments[id] = &Assignment{ID: id, AgentID: agentID, Task: task, Status: Assigned}
	return id, nil
}

func (m *StubManager) Update(id string, status Status, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.assignments[id]; ok {
		a.Status = status
		a.Result = result
	}
	return nil
}

func (m *StubManager) List(status Status) []Assignment {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Assignment
	for _, a := range m.assignments {
		if a.Status == status {
			out = append(out, *a)
		}
	}
	return out
}
