// world.go — initializes Bugle facade for Djinn's agent state management.
package persona

import (
	jAgent "github.com/dpopsuev/jericho/agent"
	jBilling "github.com/dpopsuev/jericho/billing"
	jPool "github.com/dpopsuev/jericho/pool"
	jSymbol "github.com/dpopsuev/jericho/symbol"
	"github.com/dpopsuev/jericho/world"
)

// StaffWorld wraps the Bugle facade Staff with Djinn-specific extras
// (billing tracker, color registry) that the facade doesn't own yet.
type StaffWorld struct {
	*jAgent.Staff
	Tracker  *jBilling.InMemoryTracker // billing tracker (not in facade yet)
	Registry *jSymbol.Registry         // agent color registry (not in facade yet)
}

// NewStaffWorld creates a fully wired Bugle facade for Djinn.
//
// Callers that previously accessed raw subsystems (World, Transport, Bus, Pool)
// can still reach them via the Staff escape hatches:
//
//	sw.World()     → *world.World
//	sw.Transport() → *world.LocalTransport
//	sw.Bus()       → world.Bus
//	sw.Pool()      → *world.AgentPool
func NewStaffWorld(launcher jPool.AgentSupervisor) *StaffWorld {
	staff := jAgent.NewStaff(launcher)
	return &StaffWorld{
		Staff:    staff,
		Tracker:  jBilling.NewTracker(),
		Registry: jSymbol.NewRegistry(),
	}
}

// AssignDisplay creates a Display identity for an agent using the heraldic
// color system. Registry.Assign() generates a unique color+name pair.
// The Display can be attached to an agent handle via SetDisplay().
func (sw *StaffWorld) AssignDisplay(role, scope string) (world.Display, error) {
	ci, err := sw.Registry.Assign(role, scope)
	if err != nil {
		return world.Display{}, err
	}
	return world.Display{
		Name:  ci.Short(), // heraldic color name (e.g., "Denim")
		Color: ci.Hex,     // CSS hex (e.g., "#6F8FAF")
	}, nil
}
