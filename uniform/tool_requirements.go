// tool_requirements.go — maps tool names to required capabilities (REF-77).
//
// Tools don't declare their own requirements (Battery's Tool interface is
// unchanged). Instead, ToolRequirements holds the mapping externally.
// This keeps Djinn's RBAC out of the library layer.
//
// Resolution: ToolClearance filters tools where required ⊆ agent capabilities.
package uniform

// ToolRequirements maps tool names to the capabilities required to use them.
// Tools not in the map are unrestricted (require no capabilities).
type ToolRequirements struct {
	reqs map[string][]Capability
}

// NewToolRequirements creates an empty requirements map.
func NewToolRequirements() *ToolRequirements {
	return &ToolRequirements{reqs: make(map[string][]Capability)}
}

// Set declares which capabilities a tool requires.
func (r *ToolRequirements) Set(toolName string, caps ...Capability) {
	r.reqs[toolName] = caps
}

// Requires returns the capabilities required to use a tool.
// Returns nil for tools with no requirements (unrestricted).
func (r *ToolRequirements) Requires(toolName string) []Capability {
	return r.reqs[toolName]
}

// Allowed checks if a tool's requirements are satisfied by the given capabilities.
// Tools with no requirements are always allowed.
func (r *ToolRequirements) Allowed(toolName string, agentCaps []Capability) bool {
	required := r.reqs[toolName]
	if len(required) == 0 {
		return true // unrestricted
	}

	capSet := make(map[Capability]bool, len(agentCaps))
	for _, c := range agentCaps {
		capSet[c] = true
	}

	for _, req := range required {
		if !capSet[req] {
			return false
		}
	}
	return true
}

// Filter returns only the tool names that are allowed given the agent's capabilities.
func (r *ToolRequirements) Filter(toolNames []string, agentCaps []Capability) []string {
	capSet := make(map[Capability]bool, len(agentCaps))
	for _, c := range agentCaps {
		capSet[c] = true
	}

	var allowed []string
	for _, name := range toolNames {
		required := r.reqs[name]
		if len(required) == 0 {
			allowed = append(allowed, name)
			continue
		}
		ok := true
		for _, req := range required {
			if !capSet[req] {
				ok = false
				break
			}
		}
		if ok {
			allowed = append(allowed, name)
		}
	}
	return allowed
}

// DefaultToolRequirements returns the built-in tool→capability mapping from REF-77.
// MCP tools use the same mapping — tool name is the raw name (e.g. "artifact", not "mcp__scribe__artifact").
func DefaultToolRequirements() *ToolRequirements {
	r := NewToolRequirements()

	// read capability
	r.Set("Read", CapRead)
	r.Set("Glob", CapRead)
	r.Set("Grep", CapRead)

	// write capability
	r.Set("Write", CapWrite)
	r.Set("Edit", CapWrite)

	// code capability
	r.Set("Symbol", CapCode)
	r.Set("Build", CapCode)
	r.Set("Test", CapCode)
	r.Set("Lint", CapCode)
	r.Set("arch", CapCode)

	// vcs capability
	r.Set("git", CapVCS)

	// observe capability
	r.Set("observe", CapObserve)
	r.Set("latency", CapObserve)
	r.Set("reconcile", CapObserve)

	// coordinate capability
	r.Set("assignment", CapCoordinate)

	// work capability (universal — all agents have this)
	r.Set("plan", CapWork)

	// shell capability
	r.Set("Bash", CapShell)

	// communicate capability (universal — all agents have this)
	r.Set("discourse", CapCommunicate)
	r.Set("scratch_paper", CapCommunicate)
	r.Set("render", CapCommunicate)

	return r
}
