package bugleport

import "github.com/dpopsuev/bugle/facade"

// Facade type aliases — API for Humans layer over ECS internals.
type (
	Staff       = facade.Staff
	AgentHandle = facade.AgentHandle
)

// Facade constructor.
var NewStaff = facade.NewStaff
