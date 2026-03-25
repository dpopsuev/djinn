package bugleport

import "github.com/dpopsuev/bugle/identity"

// Type aliases — definitions live in bugle/identity.
type (
	AgentIdentity   = identity.AgentIdentity
	Role            = identity.Role
	ModelIdentity   = identity.ModelIdentity
	Persona         = identity.Persona
	PersonaResolver = identity.PersonaResolver
)

// Role constants.
const (
	RoleWorker   = identity.RoleWorker
	RoleManager  = identity.RoleManager
	RoleEnforcer = identity.RoleEnforcer
	RoleBroker   = identity.RoleBroker
)

// DefaultPersonaResolver resolves personas by name.
var DefaultPersonaResolver = identity.DefaultPersonaResolver
