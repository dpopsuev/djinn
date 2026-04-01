// world.go — initializes Bugle facade for Djinn's agent state management.
package staff

import "github.com/dpopsuev/djinn/jerichoport"

// StaffWorld wraps the Bugle facade Staff with Djinn-specific extras
// (billing tracker, color registry) that the facade doesn't own yet.
type StaffWorld struct {
	*jerichoport.Staff
	Tracker  *jerichoport.InMemoryTracker // billing tracker (not in facade yet)
	Registry *jerichoport.Registry        // agent color registry (not in facade yet)
}

// NewStaffWorld creates a fully wired Bugle facade for Djinn.
//
// Callers that previously accessed raw subsystems (World, Transport, Bus, Pool)
// can still reach them via the Staff escape hatches:
//
//	sw.World()     → *jerichoport.World
//	sw.Transport() → *jerichoport.LocalTransport
//	sw.Bus()       → jerichoport.Bus
//	sw.Pool()      → *jerichoport.AgentPool
func NewStaffWorld(launcher jerichoport.AgentSupervisor) *StaffWorld {
	staff := jerichoport.NewStaff(launcher)
	return &StaffWorld{
		Staff:    staff,
		Tracker:  jerichoport.NewTracker(),
		Registry: jerichoport.NewRegistry(),
	}
}

// AssignDisplay creates a Display identity for an agent using the heraldic
// color system. Registry.Assign() generates a unique color+name pair.
// The Display can be attached to an agent handle via SetDisplay().
func (sw *StaffWorld) AssignDisplay(role, scope string) (jerichoport.Display, error) {
	ci, err := sw.Registry.Assign(role, scope)
	if err != nil {
		return jerichoport.Display{}, err
	}
	return jerichoport.Display{
		Name:  ci.Short(), // heraldic color name (e.g., "Denim")
		Color: ci.Hex,     // CSS hex (e.g., "#6F8FAF")
	}, nil
}
