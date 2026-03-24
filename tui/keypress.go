// keypress.go — priority-based key routing for the TUI.
// Higher priority handlers consume keys before lower ones.
// Approval=Critical, TabComplete=High, FocusCycle=Normal, Input=Low.
package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

// KeyPriority determines handler execution order.
type KeyPriority int

const (
	PriorityLow      KeyPriority = -100
	PriorityNormal   KeyPriority = 0
	PriorityHigh     KeyPriority = 100
	PriorityCritical KeyPriority = 200
)

// KeyHandler is a prioritized key event handler.
// Handle returns (consumed, cmd). If consumed=true, lower-priority handlers are skipped.
type KeyHandler struct {
	Name     string
	Priority KeyPriority
	Handle   func(tea.KeyMsg) (bool, tea.Cmd)
}

// KeyRouter dispatches key events through prioritized handlers.
type KeyRouter struct {
	handlers []KeyHandler
	sorted   bool
}

// NewKeyRouter creates an empty key router.
func NewKeyRouter() *KeyRouter {
	return &KeyRouter{}
}

// Register adds a handler. Call before Handle().
func (r *KeyRouter) Register(h KeyHandler) {
	r.handlers = append(r.handlers, h)
	r.sorted = false
}

// Handle dispatches a key event through handlers in priority order (highest first).
// Returns (consumed, cmd). If no handler consumes the key, returns (false, nil).
func (r *KeyRouter) Handle(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !r.sorted {
		sort.Slice(r.handlers, func(i, j int) bool {
			return r.handlers[i].Priority > r.handlers[j].Priority
		})
		r.sorted = true
	}
	for _, h := range r.handlers {
		if consumed, cmd := h.Handle(msg); consumed {
			return true, cmd
		}
	}
	return false, nil
}
