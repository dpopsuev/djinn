// Package staff implements the Djinn staffing model — role definitions,
// deterministic scheduler, and per-role memory isolation.
// Everything is YAML-driven. No hardcoded roles or tool capabilities.
package staff

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Sentinel errors for staff config validation.
var ErrUnknownCapability = errors.New("references unknown capability")

// Role defines a staff role. Loaded from YAML, not hardcoded.
type Role struct {
	Name             string   `yaml:"name"`
	Prompt           string   `yaml:"prompt"`            // path to prompt file OR inline text
	Mode             string   `yaml:"mode"`              // ask, plan, agent, auto
	ToolCapabilities []string `yaml:"tool_capabilities"` // explicit whitelist — empty means NO capabilities
	Model            string   `yaml:"model,omitempty"`   // preferred model (empty = default)
}

// ToolCapability defines a named capability backed by an MCP server or built-in.
// Part of the ToolArsenal — the full collection of tools issued to agents.
type ToolCapability struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Backend     string   `yaml:"backend"`         // MCP server name or "builtin"
	Tools       []string `yaml:"tools,omitempty"` // which tools from that backend
}

// StaffConfig is the top-level YAML config for roles and tool capabilities.
type StaffConfig struct {
	Roles            []Role              `yaml:"roles"`
	ToolCapabilities []ToolCapability    `yaml:"tool_capabilities"`
	ToolCategories   map[string][]string `yaml:"tool_categories,omitempty"`
}

// LoadConfig reads a staff config from a YAML file.
func LoadConfig(path string) (*StaffConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read staff config: %w", err)
	}
	var cfg StaffConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse staff config: %w", err)
	}
	// Resolve prompt file references.
	dir := filepath.Dir(path)
	for i := range cfg.Roles {
		cfg.Roles[i].Prompt = resolvePrompt(dir, cfg.Roles[i].Prompt)
	}
	return &cfg, nil
}

// RoleMap converts a StaffConfig's roles to a map keyed by name.
func (c *StaffConfig) RoleMap() map[string]Role {
	m := make(map[string]Role, len(c.Roles))
	for _, r := range c.Roles {
		m[r.Name] = r
	}
	return m
}

// ToolCapabilityMap converts a StaffConfig's tool capabilities to a map keyed by name.
func (c *StaffConfig) ToolCapabilityMap() map[string]ToolCapability {
	m := make(map[string]ToolCapability, len(c.ToolCapabilities))
	for _, s := range c.ToolCapabilities {
		m[s.Name] = s
	}
	return m
}

// RoleNames returns sorted role names from config.
func (c *StaffConfig) RoleNames() []string {
	names := make([]string, len(c.Roles))
	for i, r := range c.Roles {
		names[i] = r.Name
	}
	sort.Strings(names)
	return names
}

// ResolveToolNames converts capability names to the actual tool names they expose.
// Used to build CapabilityToken.AllowedTools from a role's capability list.
func (c *StaffConfig) ResolveToolNames(capNames []string) []string {
	capMap := c.ToolCapabilityMap()
	var tools []string
	for _, name := range capNames {
		if tc, ok := capMap[name]; ok {
			for _, toolName := range tc.Tools {
				tools = append(tools, toolName)
				// Also include MCP-prefixed form.
				if tc.Backend != "" && tc.Backend != "builtin" {
					tools = append(tools, fmt.Sprintf("mcp__%s__%s", tc.Backend, toolName))
				}
			}
		}
	}
	return tools
}

// Validate checks that all role capability references and category references
// point to real capabilities. Returns an error on the first violation.
func (c *StaffConfig) Validate() error {
	capMap := c.ToolCapabilityMap()
	for _, role := range c.Roles {
		for _, capName := range role.ToolCapabilities {
			if _, ok := capMap[capName]; !ok {
				return fmt.Errorf("role %q %w: %q", role.Name, ErrUnknownCapability, capName)
			}
		}
	}
	for stage, caps := range c.ToolCategories {
		for _, capName := range caps {
			if _, ok := capMap[capName]; !ok {
				return fmt.Errorf("category %q %w: %q", stage, ErrUnknownCapability, capName)
			}
		}
	}
	return nil
}

// MergeConfig overlays one StaffConfig onto a base. Overlay roles replace
// base roles by name. Overlay capabilities replace by name. Categories merge.
func MergeConfig(base, overlay *StaffConfig) *StaffConfig {
	result := &StaffConfig{
		Roles:            make([]Role, len(base.Roles)),
		ToolCapabilities: make([]ToolCapability, len(base.ToolCapabilities)),
		ToolCategories:   make(map[string][]string),
	}
	copy(result.Roles, base.Roles)
	copy(result.ToolCapabilities, base.ToolCapabilities)
	for k, v := range base.ToolCategories {
		result.ToolCategories[k] = v
	}

	// Overlay roles replace by name.
	for _, or := range overlay.Roles {
		found := false
		for i, br := range result.Roles {
			if br.Name == or.Name {
				result.Roles[i] = or
				found = true
				break
			}
		}
		if !found {
			result.Roles = append(result.Roles, or)
		}
	}

	// Overlay capabilities replace by name.
	for _, oc := range overlay.ToolCapabilities {
		found := false
		for i, bc := range result.ToolCapabilities {
			if bc.Name == oc.Name {
				result.ToolCapabilities[i] = oc
				found = true
				break
			}
		}
		if !found {
			result.ToolCapabilities = append(result.ToolCapabilities, oc)
		}
	}

	// Overlay categories merge.
	for k, v := range overlay.ToolCategories {
		result.ToolCategories[k] = v
	}

	return result
}

// LoadConfigChain loads the built-in defaults and merges overlays from
// each path that exists. Later paths take priority.
func LoadConfigChain(paths ...string) *StaffConfig {
	base := DefaultConfig()
	for _, path := range paths {
		overlay, err := LoadConfig(path)
		if err != nil {
			continue
		}
		base = MergeConfig(base, overlay)
	}
	return base
}

// resolvePrompt loads a prompt from file if it looks like a path, otherwise returns as-is.
func resolvePrompt(baseDir, prompt string) string {
	if prompt == "" {
		return ""
	}
	// If it contains newlines, it's inline text.
	for _, c := range prompt {
		if c == '\n' {
			return prompt
		}
	}
	// Try as file path (relative to config dir).
	path := prompt
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return prompt // not a file — treat as inline text
	}
	return string(data)
}

// DefaultConfig returns the built-in staff config used when no YAML is provided.
func DefaultConfig() *StaffConfig {
	return &StaffConfig{
		Roles: []Role{
			{Name: "gensec", Prompt: defaultPrompt("gensec"), Mode: "plan",
				ToolCapabilities: []string{"WorkTracking", "IssueTracking", "SignalBroadcasting", "MemoryRecall"}},
			{Name: "auditor", Prompt: defaultPrompt("auditor"), Mode: "plan",
				ToolCapabilities: []string{"WorkTracking", "RuleResolution", "MemoryRecall"}},
			{Name: "scheduler", Prompt: defaultPrompt("scheduler"), Mode: "plan",
				ToolCapabilities: []string{"WorkTracking", "ArchitectureAnalysis", "MemoryRecall"}},
			{Name: "executor", Prompt: defaultPrompt("executor"), Mode: "agent",
				ToolCapabilities: []string{"WorkTracking", "FileEditing", "ShellExecution", "FileSearching", "QualityGating", "MemoryRecall", "WebSearching"}},
			{Name: "inspector", Prompt: defaultPrompt("inspector"), Mode: "plan",
				ToolCapabilities: []string{"FileEditing", "QualityGating", "ArchitectureAnalysis", "RuleResolution", "MemoryRecall"}},
		},
		ToolCapabilities: []ToolCapability{
			{Name: "WorkTracking", Description: "Create, list, update work artifacts and dependency graphs", Backend: "scribe", Tools: []string{"artifact", "graph", "admin"}},
			{Name: "IssueTracking", Description: "Issue management across platforms", Backend: "emcee", Tools: []string{"emcee", "emcee_manage"}},
			{Name: "RuleResolution", Description: "Resolve coding rules and conventions", Backend: "lex", Tools: []string{"lexicon"}},
			{Name: "ArchitectureAnalysis", Description: "Architecture analysis, coupling, and dependency graphs", Backend: "locus", Tools: []string{"codograph", "analysis"}},
			{Name: "FileEditing", Description: "Read, write, and edit source files in the workspace", Backend: "builtin", Tools: []string{"Read", "Write", "Edit"}},
			{Name: "ShellExecution", Description: "Execute shell commands", Backend: "builtin", Tools: []string{"Bash"}},
			{Name: "FileSearching", Description: "Find files by pattern and search content by regex", Backend: "builtin", Tools: []string{"Glob", "Grep"}},
			{Name: "QualityGating", Description: "CI health, test results, and quality checks", Backend: "limes", Tools: []string{"run", "status", "results"}},
			{Name: "SignalBroadcasting", Description: "Emit and receive pipeline signals", Backend: "builtin"},
			{Name: "MemoryRecall", Description: "Search and retrieve conversation history", Backend: "builtin"},
			{Name: "WebSearching", Description: "Web search and page retrieval", Backend: "gundog"},
		},
		ToolCategories: map[string][]string{
			"plan":    {"WorkTracking", "IssueTracking", "RuleResolution", "MemoryRecall"},
			"code":    {"ArchitectureAnalysis", "FileEditing", "FileSearching", "WebSearching"},
			"build":   {"ShellExecution"},
			"test":    {"QualityGating"},
			"release": {},
			"deploy":  {},
			"operate": {},
			"monitor": {"SignalBroadcasting"},
		},
	}
}

func defaultPrompt(role string) string {
	prompts := map[string]string{
		"gensec":    "You are the General Secretary — second in command. Respond conversationally to greetings and questions. Only capture needs when the operator expresses one (I need X, fix Y, add Z). You do NOT write code or call tools.",
		"auditor":   "You are the Auditor — adversarial quality gate. Review artifacts, find gaps, challenge assumptions. Stamp or return. You do NOT write code.",
		"scheduler": "You are the Scheduler — work planner. Read specs, analyze coupling, construct task batches with dependency edges. You do NOT write code.",
		"executor":  "You are the Executor — the one who writes code. Implement the assigned task, follow acceptance criteria, write tests, signal done when complete.",
		"inspector": "You are the Inspector — quality reviewer. Review implementation against spec and architecture. Approve or reject with specific feedback. You do NOT write code.",
	}
	if p, ok := prompts[role]; ok {
		return p
	}
	return "You are the " + role + "."
}
