package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configFileName = "mcp.json"
)

// Sentinel errors.
var (
	ErrNoServers         = errors.New("no MCP servers configured")
	ErrUnknownServerType = errors.New("unknown server type")
)

// claudeConfigFile is the top-level structure Claude Code expects
// from --mcp-config.
type claudeConfigFile struct {
	MCPServers map[string]claudeServerEntry `json:"mcpServers"`
}

// claudeServerEntry represents a single MCP server in Claude Code's config.
type claudeServerEntry struct {
	URL     string            `json:"url,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	CWD     string            `json:"cwd,omitempty"`
}

// GenerateConfig produces a Claude Code MCP config JSON from a list of servers.
func GenerateConfig(servers []Server) ([]byte, error) {
	if len(servers) == 0 {
		return nil, ErrNoServers
	}

	cfg := claudeConfigFile{
		MCPServers: make(map[string]claudeServerEntry, len(servers)),
	}

	for _, s := range servers {
		entry := claudeServerEntry{}
		switch s.Type {
		case TypeHTTP:
			entry.URL = s.URL
		case TypeStdio:
			entry.Command = s.Command
			entry.Args = s.Args
			if len(s.Env) > 0 {
				entry.Env = s.Env
			}
			if s.CWD != "" {
				entry.CWD = s.CWD
			}
		default:
			return nil, fmt.Errorf("%w: %q for %q", ErrUnknownServerType, s.Type, s.Name)
		}
		cfg.MCPServers[s.Name] = entry
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// WriteConfigFile writes an MCP config to a file in the given directory.
// Returns the full path to the written file.
func WriteConfigFile(dir string, servers []Server) (string, error) {
	data, err := GenerateConfig(servers)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write mcp config: %w", err)
	}

	return path, nil
}

// DefaultServers returns the standard Aeon ecosystem MCP servers.
// These are the tools available to agents inside Misbah containers.
func DefaultServers() []Server {
	return []Server{
		{Name: "scribe", Type: TypeHTTP, URL: "http://localhost:8080/"},
		{Name: "lex", Type: TypeStdio, Command: "lex", Args: []string{"serve"}},
		{Name: "emcee", Type: TypeStdio, Command: "emcee", Args: []string{"serve"}},
	}
}
