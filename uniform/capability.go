// capability.go — RBAC three-entity model (REF-77, DJN-GOL-96).
//
// Three entities: Role HAS capabilities, Tool REQUIRES capabilities.
// ToolClearance filters where requires ⊆ capabilities.
//
// 9 capabilities, 7 composable roles, agent base = [communicate, work].
// Roles compose other roles (recursive set union). No inheritance.
package uniform

// Capability represents a permission. Open type — any string is valid.
// The 9 well-known values from REF-77 are defined as consts below.
// Custom capabilities (e.g. Capability("deploy")) are first-class.
type Capability string

const (
	CapRead        Capability = "read"        // Read, Glob, Grep
	CapWrite       Capability = "write"       // Write, Edit
	CapCode        Capability = "code"        // Symbol, Build, Test, Lint
	CapVCS         Capability = "vcs"         // VCS (git)
	CapObserve     Capability = "observe"     // Observe
	CapCoordinate  Capability = "coordinate"  // Assignment
	CapWork        Capability = "work"        // Task (pull)
	CapShell       Capability = "shell"       // Bash
	CapCommunicate Capability = "communicate" // Discourse, Notes
)

// BuiltinCapabilities returns the 9 well-known capabilities from REF-77.
// Custom capabilities can be created as Capability("deploy"), Capability("audit"), etc.
// The type is an open string — these consts are convenience, not a closed enum.
func BuiltinCapabilities() []Capability {
	return []Capability{
		CapRead, CapWrite, CapCode, CapVCS,
		CapObserve, CapCoordinate, CapWork,
		CapShell, CapCommunicate,
	}
}

// RoleDef defines a composable role. Roles compose other roles via Composes.
// Resolution: persona → role stack → recursive set union → capabilities.
type RoleDef struct {
	Name         string       `yaml:"name"`
	Composes     []string     `yaml:"composes,omitempty"`     // other role names this role includes
	Capabilities []Capability `yaml:"capabilities,omitempty"` // additional capabilities beyond composed roles
}

// RoleRegistry holds all role definitions and resolves them.
type RoleRegistry struct {
	roles map[string]RoleDef
}

// NewRoleRegistry creates a registry from role definitions.
func NewRoleRegistry(defs []RoleDef) *RoleRegistry {
	m := make(map[string]RoleDef, len(defs))
	for _, d := range defs {
		m[d.Name] = d
	}
	return &RoleRegistry{roles: m}
}

// Resolve returns the full set of capabilities for a role,
// recursively composing all parent roles (set union).
// Returns nil for unknown roles. Detects cycles.
func (r *RoleRegistry) Resolve(roleName string) []Capability {
	seen := make(map[string]bool)
	caps := make(map[Capability]bool)
	r.resolve(roleName, seen, caps)

	result := make([]Capability, 0, len(caps))
	for c := range caps {
		result = append(result, c)
	}
	return result
}

func (r *RoleRegistry) resolve(name string, seen map[string]bool, caps map[Capability]bool) {
	if seen[name] {
		return // cycle protection
	}
	seen[name] = true

	def, ok := r.roles[name]
	if !ok {
		return // unknown role
	}

	// Add this role's own capabilities.
	for _, c := range def.Capabilities {
		caps[c] = true
	}

	// Recursively compose parent roles.
	for _, parent := range def.Composes {
		r.resolve(parent, seen, caps)
	}
}

// ResolvePersona resolves a persona's stacked roles into a unified capability set.
// A persona can have multiple roles (e.g. GenSec = [director, manager]).
func (r *RoleRegistry) ResolvePersona(roleNames []string) []Capability {
	caps := make(map[Capability]bool)
	seen := make(map[string]bool)
	for _, name := range roleNames {
		r.resolve(name, seen, caps)
	}

	result := make([]Capability, 0, len(caps))
	for c := range caps {
		result = append(result, c)
	}
	return result
}

// HasCapability checks if a resolved capability set includes a specific capability.
func HasCapability(caps []Capability, target Capability) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}

// DefaultRoles returns the 7 composable roles + agent base from REF-77.
func DefaultRoles() []RoleDef {
	return []RoleDef{
		{Name: "agent", Capabilities: []Capability{CapCommunicate, CapWork}},
		{Name: "developer", Composes: []string{"agent"}, Capabilities: []Capability{CapRead, CapWrite, CapCode, CapVCS}},
		{Name: "architect", Composes: []string{"agent"}, Capabilities: []Capability{CapRead, CapCode, CapObserve, CapCoordinate}},
		{Name: "qa", Composes: []string{"agent"}, Capabilities: []Capability{CapRead, CapCode}},
		{Name: "operations", Composes: []string{"agent"}, Capabilities: []Capability{CapRead, CapObserve}},
		{Name: "manager", Composes: []string{"agent"}, Capabilities: []Capability{CapObserve, CapCoordinate}},
		{Name: "director", Composes: []string{"manager"}, Capabilities: []Capability{CapShell}},
		{Name: "operator", Composes: []string{"developer", "architect", "qa", "operations"}, Capabilities: []Capability{CapShell}},
	}
}
