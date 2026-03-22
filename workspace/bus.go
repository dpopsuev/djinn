package workspace

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/dpopsuev/djinn/djinnlog"
)

// EventType identifies workspace lifecycle events.
type EventType int

const (
	EventSwitch      EventType = iota // entire workspace changed
	EventRepoAdd                      // repo added to workspace
	EventRepoRemove                   // repo removed from workspace
	EventScopeChange                  // per-agent view changed (multi-agent)
)

var eventNames = [...]string{"switch", "repo.add", "repo.remove", "scope.change"}

func (t EventType) String() string {
	if int(t) < len(eventNames) {
		return eventNames[t]
	}
	return fmt.Sprintf("EventType(%d)", t)
}

// Event is a workspace lifecycle event.
type Event struct {
	Type    EventType
	Old     *Workspace // nil on first load
	New     *Workspace
	AgentID string // empty = broadcast, set = scoped to agent
	Repo    *Repo  // set for RepoAdd/RepoRemove
}

// Handler reacts to a workspace event.
type Handler struct {
	Name string
	Fn   func(Event)
}

// Bus dispatches workspace events to subscribers.
// Each handler runs independently — a panic in one doesn't block others.
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
	log      *slog.Logger
}

// NewBus creates a workspace event bus.
func NewBus(log *slog.Logger) *Bus {
	if log == nil {
		log = djinnlog.Nop()
	}
	return &Bus{log: log}
}

// On registers a named handler for workspace events.
func (b *Bus) On(name string, fn func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, Handler{Name: name, Fn: fn})
}

// Emit dispatches an event to all handlers.
// Each handler is called sequentially. Panics are recovered and logged.
func (b *Bus) Emit(evt Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	b.log.Info("workspace event", "type", evt.Type.String())

	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					b.log.Error("workspace handler panic",
						"handler", h.Name,
						"type", evt.Type.String(),
						"panic", fmt.Sprint(r),
					)
				}
			}()
			h.Fn(evt)
			b.log.Debug("workspace handler completed", "handler", h.Name, "type", evt.Type.String())
		}()
	}
}

// HandlerCount returns the number of registered handlers.
func (b *Bus) HandlerCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers)
}
