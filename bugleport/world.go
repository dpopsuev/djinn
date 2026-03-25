package bugleport

import (
	"github.com/dpopsuev/bugle/palette"
	"github.com/dpopsuev/bugle/world"
)

// Type aliases — definitions live in bugle/world.
type (
	World         = world.World
	EntityID      = world.EntityID
	Component     = world.Component
	ComponentType = world.ComponentType
	DiffKind      = world.DiffKind
	DiffHook      = world.DiffHook
	AgentState    = world.AgentState
	Health        = world.Health
	Hierarchy     = world.Hierarchy
	Budget        = world.Budget
	Progress      = world.Progress
)

// Agent state constants.
const (
	Active  = world.Active
	Idle    = world.Idle
	Stale   = world.Stale
	Errored = world.Errored
	Done    = world.Done
)

// Generic wrappers.
func Attach[T world.Component](w *world.World, id world.EntityID, c T) { world.Attach(w, id, c) }
func Get[T world.Component](w *world.World, id world.EntityID) T       { return world.Get[T](w, id) }
func TryGet[T world.Component](w *world.World, id world.EntityID) (T, bool) {
	return world.TryGet[T](w, id)
}

// Constructor.
var NewWorld = world.NewWorld

// Palette — color identity for visual agent badges.
type (
	ColorIdentity = palette.ColorIdentity
	Registry      = palette.Registry
)

var NewRegistry = palette.NewRegistry
