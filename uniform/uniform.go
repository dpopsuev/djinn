// uniform.go — Uniform is the resolved spawn config for an agent (TSK-748, REF-77).
//
// Persona → Roles → Capabilities (set union) → Tools (requires ⊆ capabilities).
// Uniform is immutable after construction. Built by NewUniform at spawn time.
// The agent sees only the tools in its Uniform. System prompt lists only these tools.
package uniform

import "strings"

// Uniform is the resolved spawn config for an agent. Immutable after creation.
// Contains the persona name, resolved capabilities, and filtered tool list.
type Uniform struct {
	persona      string
	roles        []string
	capabilities []Capability
	tools        []string // tool names the agent can see and use
	denied       []string // tool names filtered out by RBAC
	mode         string   // ask, plan, agent, auto
	model        string   // preferred LLM model
	prompt       string   // system prompt text
	warnings     []string // resolution warnings (ORANGE signals)
}

// NewUniform resolves a persona into a fully wired Uniform.
// Resolves: persona roles → capabilities (set union) → filter tools by requires.
func NewUniform(
	persona string,
	roles []string,
	registry *RoleRegistry,
	requirements *ToolRequirements,
	availableTools []string,
	mode string,
	model string,
	prompt string,
) *Uniform {
	caps := registry.ResolvePersona(roles)
	tools := requirements.Filter(availableTools, caps)

	// Compute denied tools and warnings.
	var denied []string
	deniedSet := make(map[string]bool)
	for _, t := range availableTools {
		if !requirements.Allowed(t, caps) {
			denied = append(denied, t)
			deniedSet[t] = true
		}
	}

	var warnings []string

	// ORANGE: no capabilities resolved — agent can't do anything meaningful.
	if len(caps) == 0 {
		warnings = append(warnings, "no capabilities resolved for persona "+persona+" — check role definitions")
	}

	// ORANGE: no tools after filtering — agent has capabilities but no matching tools.
	if len(caps) > 0 && len(tools) == 0 {
		warnings = append(warnings, "all tools filtered out for persona "+persona+" — no tools require only these capabilities")
	}

	// ORANGE: unknown roles (resolved to nothing).
	for _, r := range roles {
		if len(registry.Resolve(r)) == 0 {
			warnings = append(warnings, "unknown role "+r+" for persona "+persona)
		}
	}

	_ = deniedSet // used for denied list

	return &Uniform{
		persona:      persona,
		roles:        roles,
		capabilities: caps,
		tools:        tools,
		denied:       denied,
		mode:         mode,
		model:        model,
		prompt:       prompt,
		warnings:     warnings,
	}
}

// Persona returns the agent's persona name (e.g. "gensec", "coder-1").
func (u *Uniform) Persona() string { return u.persona }

// Roles returns the persona's role stack (e.g. ["director", "manager"]).
func (u *Uniform) Roles() []string { return u.roles }

// Capabilities returns the resolved capability set (set union of all roles).
func (u *Uniform) Capabilities() []Capability { return u.capabilities }

// Tools returns the filtered tool names this agent can see and use.
func (u *Uniform) Tools() []string { return u.tools }

// Mode returns the agent's operating mode (ask, plan, agent, auto).
func (u *Uniform) Mode() string { return u.mode }

// Model returns the preferred LLM model for this agent.
func (u *Uniform) Model() string { return u.model }

// Prompt returns the system prompt text for this agent.
func (u *Uniform) Prompt() string { return u.prompt }

// Denied returns tool names that were filtered out by RBAC.
func (u *Uniform) Denied() []string { return u.denied }

// Warnings returns resolution warnings (ORANGE signals).
// Empty means clean resolution. Callers should log these as slog.Warn.
func (u *Uniform) Warnings() []string { return u.warnings }

// HasCapability checks if this agent has a specific capability.
func (u *Uniform) HasCapability(c Capability) bool {
	return HasCapability(u.capabilities, c)
}

// HasTool checks if a tool is in this agent's allowed set.
func (u *Uniform) HasTool(name string) bool {
	for _, t := range u.tools {
		if t == name {
			return true
		}
	}
	return false
}

// SystemContext returns a prompt fragment describing this agent's identity
// and available tools. Append to the system prompt at spawn time.
// Lists ONLY the tools in this Uniform — no mention of unavailable tools.
func (u *Uniform) SystemContext() string {
	if u.prompt == "" && len(u.tools) == 0 {
		return ""
	}

	var b strings.Builder
	if u.prompt != "" {
		b.WriteString(u.prompt)
		b.WriteString("\n\n")
	}

	if len(u.tools) > 0 {
		b.WriteString("You have access to these tools: ")
		for i, t := range u.tools {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(t)
		}
		b.WriteString(".\nDo not attempt to use tools not listed above.")
	}

	return b.String()
}
