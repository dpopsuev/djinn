// clearance.go — ToolClearance determines which tools a role has clearance to use.
//
// The agent sees tool names like "Read", "Bash", "Glob".
// ToolClearance checks if the tool is in the current role's allowed capabilities.
// If yes, it forwards to the underlying registry. If no, permission denied.
//
// This is the ToolArsenal's runtime enforcement layer. The capability definitions
// in staff.yaml declare WHICH tools each role can see. ToolClearance
// enforces it at call time.
//
// ToolClearance implements the same Execute/All interface as builtin.Registry
// so the agent loop doesn't change — it just gets a filtered view.
package staff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/tools/builtin"
)

// Sentinel errors for ToolClearance.
var ErrToolNotAllowed = errors.New("tool not available for role")

// ToolClearance wraps a tool registry with role-based capability filtering.
// The agent sees only the tools that belong to the current role's capabilities.
type ToolClearance struct {
	mu       sync.RWMutex
	registry *builtin.Registry
	config   *StaffConfig
	role     string // current active role

	// Resolved: which raw tool names are allowed for the current role.
	// Built on SetRole() from role.ToolCapabilities → capability.Tools mapping.
	allowed map[string]bool
}

// NewToolClearance creates a clearance filter that restricts tools by role capabilities.
func NewToolClearance(cfg *StaffConfig, registry *builtin.Registry, initialRole string) *ToolClearance {
	r := &ToolClearance{
		registry: registry,
		config:   cfg,
		allowed:  make(map[string]bool),
	}
	r.SetRole(initialRole)
	return r
}

// SetRole changes which tools are visible to the agent.
// Resolves the role's capability names to actual tool names via the capability config.
func (r *ToolClearance) SetRole(role string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.role = role
	r.allowed = make(map[string]bool)

	roleDef, ok := r.config.RoleMap()[role]
	if !ok {
		return // unknown role = no tools
	}

	capMap := r.config.ToolCapabilityMap()
	for _, capName := range roleDef.ToolCapabilities {
		tc, ok := capMap[capName]
		if !ok {
			continue
		}
		// Each capability declares which tools from its backend are exposed.
		for _, toolName := range tc.Tools {
			r.allowed[toolName] = true
			// Also allow the MCP-prefixed form: mcp__<backend>__<tool>
			if tc.Backend != "" && tc.Backend != "builtin" {
				r.allowed[fmt.Sprintf("mcp__%s__%s", tc.Backend, toolName)] = true
			}
		}
	}
}

// Execute dispatches a tool call, but only if the tool is in the
// current role's allowed set. Returns permission denied otherwise.
func (r *ToolClearance) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	r.mu.RLock()
	allowed := r.allowed[name]
	r.mu.RUnlock()

	if !allowed {
		return "", fmt.Errorf("%w: %q for %q", ErrToolNotAllowed, name, r.role)
	}
	return r.registry.Execute(ctx, name, input)
}

// All returns only the tools visible to the current role.
func (r *ToolClearance) All() []builtin.Tool {
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
func (r *ToolClearance) Names() []string {
	tools := r.All()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name()
	}
	return names
}

// Role returns the current active role name.
func (r *ToolClearance) Role() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.role
}
