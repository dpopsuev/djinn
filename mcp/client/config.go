package client

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServerConfig defines an MCP server to connect to.
type ServerConfig struct {
	Command string            `json:"command,omitempty" yaml:"command"`
	Args    []string          `json:"args,omitempty" yaml:"args"`
	URL     string            `json:"url,omitempty" yaml:"url"`
	Env     map[string]string `json:"env,omitempty" yaml:"env"`
}

// IsHTTP returns true if this server uses HTTP transport.
func (c ServerConfig) IsHTTP() bool {
	return c.URL != ""
}

// LoadMCPConfig loads MCP server configs.
// If workspaceMCP is non-empty, it's used exclusively (workspace owns MCP config).
// Otherwise falls back to djinn.yaml + cursor/claude config.
func LoadMCPConfig(workdir string, claudeHome string, workspaceMCP ...map[string]ServerConfig) map[string]ServerConfig {
	// Workspace MCP section = exclusive source (no merging with cursor)
	if len(workspaceMCP) > 0 && len(workspaceMCP[0]) > 0 {
		servers := make(map[string]ServerConfig)
		for k, v := range workspaceMCP[0] {
			servers[k] = v
		}
		return servers
	}

	servers := make(map[string]ServerConfig)

	// 1. Claude Code: ~/.cursor/mcp.json (lower priority)
	loadCursorMCP(filepath.Join(os.Getenv("HOME"), ".cursor", "mcp.json"), servers)

	// 2. Claude Code: settings that might contain mcpServers
	loadClaudeSettings(filepath.Join(claudeHome, "settings.json"), servers)

	// 3. djinn.yaml mcp section (highest priority)
	loadDjinnYAML(filepath.Join(workdir, "djinn.yaml"), servers)

	return servers
}

func loadDjinnYAML(path string, servers map[string]ServerConfig) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg struct {
		MCP map[string]ServerConfig `yaml:"mcp"`
	}
	if yaml.Unmarshal(data, &cfg) != nil {
		return
	}
	for name, sc := range cfg.MCP {
		servers[name] = sc // overrides lower-priority sources
	}
}

type claudeSettingsFile struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

func loadClaudeSettings(path string, servers map[string]ServerConfig) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var settings claudeSettingsFile
	if json.Unmarshal(data, &settings) != nil {
		return
	}
	for name, sc := range settings.MCPServers {
		if _, exists := servers[name]; !exists {
			servers[name] = sc
		}
	}
}

func loadCursorMCP(path string, servers map[string]ServerConfig) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg claudeSettingsFile // same format: {"mcpServers": {...}}
	if json.Unmarshal(data, &cfg) != nil {
		return
	}
	for name, sc := range cfg.MCPServers {
		if _, exists := servers[name]; !exists {
			servers[name] = sc
		}
	}
}
