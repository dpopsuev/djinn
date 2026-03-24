// Package staff implements the Djinn staffing model — role definitions,
// deterministic scheduler, and per-role memory isolation.
// Everything is YAML-driven. No hardcoded roles or slots.
package staff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Role defines a staff role. Loaded from YAML, not hardcoded.
type Role struct {
	Name   string   `yaml:"name"`
	Prompt string   `yaml:"prompt"` // path to prompt file OR inline text
	Mode   string   `yaml:"mode"`   // ask, plan, agent, auto
	Slots  []string `yaml:"slots"`  // explicit whitelist — empty means NO slots
	Model  string   `yaml:"model,omitempty"` // preferred model (empty = default)
}

// Slot defines a Spine slot — a named capability backed by an MCP server or built-in.
type Slot struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Backend     string   `yaml:"backend"`              // MCP server name or "builtin"
	Tools       []string `yaml:"tools,omitempty"`       // which tools from that backend
}

// StaffConfig is the top-level YAML config for roles and slots.
type StaffConfig struct {
	Roles []Role `yaml:"roles"`
	Slots []Slot `yaml:"slots"`
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

// SlotMap converts a StaffConfig's slots to a map keyed by name.
func (c *StaffConfig) SlotMap() map[string]Slot {
	m := make(map[string]Slot, len(c.Slots))
	for _, s := range c.Slots {
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
				Slots: []string{"WorkTracker", "IssueTracker", "SignalBus", "MemoryRecall"}},
			{Name: "auditor", Prompt: defaultPrompt("auditor"), Mode: "plan",
				Slots: []string{"WorkTracker", "RuleResolver", "MemoryRecall"}},
			{Name: "scheduler", Prompt: defaultPrompt("scheduler"), Mode: "plan",
				Slots: []string{"WorkTracker", "ArchExplorer", "MemoryRecall"}},
			{Name: "executor", Prompt: defaultPrompt("executor"), Mode: "agent",
				Slots: []string{"WorkTracker", "CodeEditor", "Shell", "FileSearch", "QualityGate", "MemoryRecall", "WebSearch"}},
			{Name: "inspector", Prompt: defaultPrompt("inspector"), Mode: "plan",
				Slots: []string{"CodeEditor", "QualityGate", "ArchExplorer", "RuleResolver", "MemoryRecall"}},
		},
		Slots: []Slot{
			{Name: "WorkTracker", Description: "Create, list, update work artifacts", Backend: "scribe", Tools: []string{"artifact", "graph", "admin"}},
			{Name: "IssueTracker", Description: "Issue management across platforms", Backend: "emcee", Tools: []string{"emcee", "emcee_manage"}},
			{Name: "RuleResolver", Description: "Resolve coding rules and conventions", Backend: "lex", Tools: []string{"lexicon"}},
			{Name: "ArchExplorer", Description: "Architecture analysis and coupling", Backend: "locus", Tools: []string{"codograph", "analysis"}},
			{Name: "CodeEditor", Description: "Read, write, edit source files", Backend: "builtin", Tools: []string{"Read", "Write", "Edit"}},
			{Name: "Shell", Description: "Execute shell commands", Backend: "builtin", Tools: []string{"Bash"}},
			{Name: "FileSearch", Description: "Find files and search content", Backend: "builtin", Tools: []string{"Glob", "Grep"}},
			{Name: "QualityGate", Description: "CI health and quality checks", Backend: "limes", Tools: []string{"run", "status", "results"}},
			{Name: "SignalBus", Description: "Emit and receive pipeline signals", Backend: "builtin"},
			{Name: "MemoryRecall", Description: "Search and retrieve conversation history", Backend: "builtin"},
			{Name: "WebSearch", Description: "Web search and page retrieval", Backend: "gundog"},
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
