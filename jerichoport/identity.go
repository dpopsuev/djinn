package jerichoport

import "github.com/dpopsuev/jericho/symbol"

// Type aliases — definitions live in jericho/symbol.
type (
	Role            = symbol.Role
	ModelIdentity   = symbol.ModelIdentity
	Persona         = symbol.Persona
	PersonaResolver = symbol.PersonaResolver
)

// Role constants.
const (
	RoleWorker   = symbol.RoleWorker
	RoleManager  = symbol.RoleManager
	RoleEnforcer = symbol.RoleEnforcer
	RoleBroker   = symbol.RoleBroker
)

// DefaultPersonaResolver resolves personas by name.
var DefaultPersonaResolver = symbol.DefaultPersonaResolver
