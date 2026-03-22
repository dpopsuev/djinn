// Package mode defines the mode registry for agent operating modes.
// Each mode controls tool access and approval behavior.
package mode

import "github.com/dpopsuev/djinn/agent"

// Definition describes an agent mode's behavior.
type Definition struct {
	Name         string
	Description  string
	ToolsEnabled bool
	AutoApprove  bool
}

// Registry provides access to mode definitions.
type Registry interface {
	Get(name string) (Definition, bool)
	List() []Definition
}

// DefaultRegistry returns the built-in mode definitions.
func DefaultRegistry() Registry {
	return &builtinRegistry{}
}

type builtinRegistry struct{}

var builtinModes = []Definition{
	{Name: "ask", Description: "Read-only — no tool execution", ToolsEnabled: false, AutoApprove: false},
	{Name: "plan", Description: "Thinking only — tools visible but not executed", ToolsEnabled: false, AutoApprove: false},
	{Name: "agent", Description: "Tools with operator approval", ToolsEnabled: true, AutoApprove: false},
	{Name: "auto", Description: "Tools without approval", ToolsEnabled: true, AutoApprove: true},
}

func (r *builtinRegistry) Get(name string) (Definition, bool) {
	for _, d := range builtinModes {
		if d.Name == name {
			return d, true
		}
	}
	return Definition{}, false
}

func (r *builtinRegistry) List() []Definition {
	out := make([]Definition, len(builtinModes))
	copy(out, builtinModes)
	return out
}

// ToAgentMode converts a Definition name to an agent.Mode value.
func ToAgentMode(name string) (agent.Mode, error) {
	return agent.ParseMode(name)
}
