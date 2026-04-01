package jerichoport

import "github.com/dpopsuev/jericho/agent"

// Agent facade — API for Humans layer over ECS internals.
type (
	Staff = agent.Staff
	Solo  = agent.Solo
)

// Constructor.
var NewStaff = agent.NewStaff
