package staff

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Entry is a single message in role memory.
type Entry struct {
	ID        string    `json:"id"`
	Speaker   string    `json:"speaker"`   // role name that produced this
	Role      string    `json:"role"`      // user or assistant
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// RoleMemory provides per-role conversation isolation with a shared briefing channel.
type RoleMemory struct {
	mu       sync.RWMutex
	slices   map[string][]Entry // per-role conversation history
	briefing []Entry            // shared channel visible to all roles
	nextID   int
	allByID  map[string]*Entry  // index for Get()
}

// NewRoleMemory creates an empty role memory store.
func NewRoleMemory() *RoleMemory {
	return &RoleMemory{
		slices:  make(map[string][]Entry),
		allByID: make(map[string]*Entry),
	}
}

// Append adds an entry to a role's conversation slice. Returns the assigned ID.
func (m *RoleMemory) Append(role string, entry Entry) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	entry.ID = fmt.Sprintf("MSG-%04d", m.nextID)
	entry.Speaker = role
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	m.slices[role] = append(m.slices[role], entry)
	m.allByID[entry.ID] = &m.slices[role][len(m.slices[role])-1]
	return entry.ID
}

// History returns all entries for a specific role.
func (m *RoleMemory) History(role string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slices[role]
}

// Briefing returns the shared channel visible to all roles.
func (m *RoleMemory) Briefing() []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.briefing
}

// AppendBriefing adds an entry to the shared briefing channel. Returns ID.
func (m *RoleMemory) AppendBriefing(entry Entry) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	entry.ID = fmt.Sprintf("BRF-%04d", m.nextID)
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	m.briefing = append(m.briefing, entry)
	m.allByID[entry.ID] = &m.briefing[len(m.briefing)-1]
	return entry.ID
}

// Get retrieves an entry by ID across all slices and briefing.
func (m *RoleMemory) Get(id string) (Entry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.allByID[id]
	if !ok {
		return Entry{}, false
	}
	return *e, true
}

// Search finds entries matching a query substring across all roles and briefing.
func (m *RoleMemory) Search(query string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query = strings.ToLower(query)
	var results []Entry

	for _, entries := range m.slices {
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Content), query) {
				results = append(results, e)
			}
		}
	}
	for _, e := range m.briefing {
		if strings.Contains(strings.ToLower(e.Content), query) {
			results = append(results, e)
		}
	}
	return results
}

// Context returns the combined entries for a role: briefing + role history.
// This is what gets fed to the LLM on role switch.
func (m *RoleMemory) Context(role string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var ctx []Entry
	ctx = append(ctx, m.briefing...)
	ctx = append(ctx, m.slices[role]...)
	return ctx
}

// Len returns total entries across all slices and briefing.
func (m *RoleMemory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := len(m.briefing)
	for _, s := range m.slices {
		n += len(s)
	}
	return n
}
