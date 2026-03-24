// router.go — SlotRouter maps slot tool names to backend tools.
//
// The agent sees slot-qualified names like "Read", "Bash", "Glob".
// The router checks if the tool is in the current role's allowed slots.
// If yes, it forwards to the underlying registry. If no, permission denied.
//
// This is the Spine's runtime enforcement layer. The slot definitions
// in staff.yaml declare WHICH tools each role can see. The router
// enforces it at call time.
//
// The router implements the same Execute/All interface as builtin.Registry
// so the agent loop doesn't change — it just gets a filtered view.
package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/tools/builtin"
)

// SlotRouter wraps a tool registry with role-based slot filtering.
// The agent sees only the tools that belong to the current role's slots.
type SlotRouter struct {
	mu       sync.RWMutex
	registry *builtin.Registry
	config   *StaffConfig
	role     string // current active role

	// Resolved: which raw tool names are allowed for the current role.
	// Built on SetRole() from role.Slots → slot.Tools mapping.
	allowed map[string]bool
}

// NewSlotRouter creates a router that filters tools by role slots.
func NewSlotRouter(cfg *StaffConfig, registry *builtin.Registry, initialRole string) *SlotRouter {
	r := &SlotRouter{
		registry: registry,
		config:   cfg,
		allowed:  make(map[string]bool),
	}
	r.SetRole(initialRole)
	return r
}

// SetRole changes which tools are visible to the agent.
// Resolves the role's slot names to actual tool names via the slot config.
func (r *SlotRouter) SetRole(role string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.role = role
	r.allowed = make(map[string]bool)

	roleDef, ok := r.config.RoleMap()[role]
	if !ok {
		return // unknown role = no tools
	}

	slotMap := r.config.SlotMap()
	for _, slotName := range roleDef.Slots {
		slot, ok := slotMap[slotName]
		if !ok {
			continue
		}
		// Each slot declares which tools from its backend are exposed.
		for _, toolName := range slot.Tools {
			r.allowed[toolName] = true
			// Also allow the MCP-prefixed form: mcp__<backend>__<tool>
			if slot.Backend != "" && slot.Backend != "builtin" {
				r.allowed[fmt.Sprintf("mcp__%s__%s", slot.Backend, toolName)] = true
			}
		}
	}
}

// Execute dispatches a tool call, but only if the tool is in the
// current role's allowed set. Returns permission denied otherwise.
func (r *SlotRouter) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	r.mu.RLock()
	allowed := r.allowed[name]
	r.mu.RUnlock()

	if !allowed {
		return "", fmt.Errorf("tool %q not available for role %q", name, r.role)
	}
	return r.registry.Execute(ctx, name, input)
}

// All returns only the tools visible to the current role.
func (r *SlotRouter) All() []builtin.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var visible []builtin.Tool
	for _, tool := range r.registry.All() {
		if r.allowed[tool.Name()] {
			visible = append(visible, tool)
		}
	}
	return visible
}

// Names returns the names of visible tools.
func (r *SlotRouter) Names() []string {
	tools := r.All()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name()
	}
	return names
}

// Role returns the current active role name.
func (r *SlotRouter) Role() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.role
}
