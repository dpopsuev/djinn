package config

import "fmt"

// MCPServerEntry defines an MCP server connection in djinn.yaml.
type MCPServerEntry struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	URL     string            `yaml:"url"`
	Env     map[string]string `yaml:"env"`
}

// MCPConfigurable implements Configurable for MCP server settings.
// Config key: "mcp_servers"
//
// djinn.yaml example:
//
//	mcp_servers:
//	  scribe:
//	    command: "scribe"
//	    args: ["serve"]
//	  locus:
//	    url: "http://localhost:8090/"
type MCPConfigurable struct {
	Servers map[string]MCPServerEntry
}

func (c *MCPConfigurable) ConfigKey() string { return "mcp_servers" }

func (c *MCPConfigurable) Snapshot() any {
	if c.Servers == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(c.Servers))
	for name, entry := range c.Servers {
		e := make(map[string]any)
		if entry.Command != "" {
			e["command"] = entry.Command
		}
		if len(entry.Args) > 0 {
			e["args"] = entry.Args
		}
		if entry.URL != "" {
			e["url"] = entry.URL
		}
		if len(entry.Env) > 0 {
			e["env"] = entry.Env
		}
		out[name] = e
	}
	return out
}

func (c *MCPConfigurable) Apply(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("mcp_servers: %w: expected map, got %T", ErrUnexpectedType, v)
	}
	if c.Servers == nil {
		c.Servers = make(map[string]MCPServerEntry)
	}
	for name, raw := range m {
		entry, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("mcp_servers.%s: %w: expected map, got %T", name, ErrUnexpectedType, raw)
		}
		se := MCPServerEntry{}
		if cmd, ok := entry["command"].(string); ok {
			se.Command = cmd
		}
		if url, ok := entry["url"].(string); ok {
			se.URL = url
		}
		if args, ok := entry["args"]; ok {
			switch val := args.(type) {
			case []any:
				for _, a := range val {
					if s, ok := a.(string); ok {
						se.Args = append(se.Args, s)
					}
				}
			case []string:
				se.Args = val
			}
		}
		if env, ok := entry["env"]; ok {
			switch envVal := env.(type) {
			case map[string]any:
				se.Env = make(map[string]string, len(envVal))
				for k, v := range envVal {
					if s, ok := v.(string); ok {
						se.Env[k] = s
					}
				}
			case map[string]string:
				se.Env = envVal
			}
		}
		c.Servers[name] = se
	}
	return nil
}
