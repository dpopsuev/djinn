package client

import (
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

// LoadMCPConfig loads MCP server configs from djinn.yaml ONLY.
// Djinn owns the MCP config — agent CLIs get a mirror, not their own.
// No fallback to ~/.cursor/mcp.json or claude settings.
func LoadMCPConfig(workdir string) map[string]ServerConfig {
	servers := make(map[string]ServerConfig)
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
		servers[name] = sc
	}
}
