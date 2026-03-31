package jerichoport

import "github.com/dpopsuev/jericho/facade"

// Facade type aliases — API for Humans layer over ECS internals.
type (
	Staff       = facade.Staff
	AgentHandle = facade.AgentHandle
)

// Facade constructor.
var NewStaff = facade.NewStaff
