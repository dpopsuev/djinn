// hooks.go — Hook configuration for tool interception.
//
// Hooks allow users to run shell commands before and after tool execution.
// Configured via djinn.yaml hooks section or standalone hooks.yaml.
package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// HookEntry defines a single hook — a shell command and the tools it applies to.
type HookEntry struct {
	Command string   `yaml:"command"`
	Tools   []string `yaml:"tools"` // tool names or "*" for wildcard
}

// HooksConfig holds pre and post tool use hook definitions.
type HooksConfig struct {
	PreToolUse  []HookEntry `yaml:"pre_tool_use"`
	PostToolUse []HookEntry `yaml:"post_tool_use"`
}

// ParseHooks parses hook configuration from YAML bytes.
func ParseHooks(data []byte) (HooksConfig, error) {
	var cfg HooksConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return HooksConfig{}, fmt.Errorf("hooks: parse error: %w", err)
	}
	return cfg, nil
}
