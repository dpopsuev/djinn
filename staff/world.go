// world.go — initializes Bugle facade for Djinn's agent state management.
package staff

import "github.com/dpopsuev/djinn/bugleport"

// StaffWorld wraps the Bugle facade Staff with Djinn-specific extras
// (billing tracker, color registry) that the facade doesn't own yet.
type StaffWorld struct {
	*bugleport.Staff
	Tracker  *bugleport.InMemoryTracker // billing tracker (not in facade yet)
	Registry *bugleport.Registry        // agent color registry (not in facade yet)
}

// NewStaffWorld creates a fully wired Bugle facade for Djinn.
//
// Callers that previously accessed raw subsystems (World, Transport, Bus, Pool)
// can still reach them via the Staff escape hatches:
//
//	sw.World()     → *bugleport.World
//	sw.Transport() → *bugleport.LocalTransport
//	sw.Bus()       → bugleport.Bus
//	sw.Pool()      → *bugleport.AgentPool
func NewStaffWorld(launcher bugleport.Launcher) *StaffWorld {
	staff := bugleport.NewStaff(launcher)
	return &StaffWorld{
		Staff:    staff,
		Tracker:  bugleport.NewTracker(),
		Registry: bugleport.NewRegistry(),
	}
}
