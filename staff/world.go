// world.go — initializes Bugle World ECS for Djinn's agent state management.
package staff

import "github.com/dpopsuev/djinn/bugleport"

// StaffWorld holds the Bugle ECS world and support objects for Djinn.
type StaffWorld struct {
	World     *bugleport.World
	Transport *bugleport.LocalTransport
	Bus       *bugleport.MemBus
	Tracker   *bugleport.InMemoryTracker
	Registry  *bugleport.Registry
	Pool      *bugleport.AgentPool
}

// NewStaffWorld creates a fully wired Bugle world for Djinn.
func NewStaffWorld(launcher bugleport.Launcher) *StaffWorld {
	w := bugleport.NewWorld()
	t := bugleport.NewLocalTransport()
	bus := bugleport.NewMemBus()
	tracker := bugleport.NewTracker()
	registry := bugleport.NewRegistry()
	pool := bugleport.NewAgentPool(w, t, bus, launcher)

	return &StaffWorld{
		World:     w,
		Transport: t,
		Bus:       bus,
		Tracker:   tracker,
		Registry:  registry,
		Pool:      pool,
	}
}
