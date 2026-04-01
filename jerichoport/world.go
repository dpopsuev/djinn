package jerichoport

import (
	"github.com/dpopsuev/jericho/symbol"
	"github.com/dpopsuev/jericho/world"
)

// Type aliases — definitions live in jericho/world.
type (
	World         = world.World
	EntityID      = world.EntityID
	Component     = world.Component
	ComponentType = world.ComponentType
	DiffKind      = world.DiffKind
	DiffHook      = world.DiffHook
	Alive         = world.Alive
	AliveState    = world.AliveState
	Ready         = world.Ready
	ReadyReason   = world.ReadyReason
	Hierarchy     = world.Hierarchy
	Budget        = world.Budget
	Progress      = world.Progress
	Display       = world.Display
)

// Alive state constants.
const (
	AliveRunning    = world.AliveRunning
	AliveTerminated = world.AliveTerminated
)

// Ready reason constants.
const (
	ReasonIdle    = world.ReasonIdle
	ReasonStale   = world.ReasonStale
	ReasonErrored = world.ReasonErrored
)

// Generic wrappers.
func Attach[T world.Component](w *world.World, id world.EntityID, c T) { world.Attach(w, id, c) }
func Get[T world.Component](w *world.World, id world.EntityID) T       { return world.Get[T](w, id) }
func TryGet[T world.Component](w *world.World, id world.EntityID) (T, bool) {
	return world.TryGet[T](w, id)
}

// Constructor.
var NewWorld = world.NewWorld

// Color identity for visual agent badges.
type (
	Color    = symbol.Color
	Registry = symbol.Registry
)

var NewRegistry = symbol.NewRegistry
