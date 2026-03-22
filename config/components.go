package config

import "fmt"

// ModeConfig implements Configurable for the agent mode.
type ModeConfig struct {
	Mode string
}

func (c *ModeConfig) ConfigKey() string { return "mode" }
func (c *ModeConfig) Snapshot() any     { return c.Mode }
func (c *ModeConfig) Apply(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("mode: expected string, got %T", v)
	}
	switch s {
	case "ask", "plan", "agent", "auto":
		c.Mode = s
		return nil
	default:
		return fmt.Errorf("mode: unknown %q", s)
	}
}

// DriverConfigurable implements Configurable for driver settings.
// Never serializes API keys or tokens.
type DriverConfigurable struct {
	Name  string
	Model string
}

func (c *DriverConfigurable) ConfigKey() string { return "driver" }
func (c *DriverConfigurable) Snapshot() any {
	return map[string]string{
		"name":  c.Name,
		"model": c.Model,
	}
}
func (c *DriverConfigurable) Apply(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("driver: expected map, got %T", v)
	}
	if name, ok := m["name"].(string); ok {
		c.Name = name
	}
	if model, ok := m["model"].(string); ok {
		c.Model = model
	}
	return nil
}

// SessionConfigurable implements Configurable for session settings.
type SessionConfigurable struct {
	MaxTurns    int
	AutoApprove bool
	OutputMode  string
	NoPersist   bool
}

func (c *SessionConfigurable) ConfigKey() string { return "session" }
func (c *SessionConfigurable) Snapshot() any {
	return map[string]any{
		"max_turns":    c.MaxTurns,
		"auto_approve": c.AutoApprove,
		"output_mode":  c.OutputMode,
		"no_persist":   c.NoPersist,
	}
}
func (c *SessionConfigurable) Apply(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("session: expected map, got %T", v)
	}
	if mt, ok := m["max_turns"]; ok {
		switch val := mt.(type) {
		case int:
			c.MaxTurns = val
		case float64:
			c.MaxTurns = int(val)
		}
	}
	if aa, ok := m["auto_approve"].(bool); ok {
		c.AutoApprove = aa
	}
	if om, ok := m["output_mode"].(string); ok {
		c.OutputMode = om
	}
	if np, ok := m["no_persist"].(bool); ok {
		c.NoPersist = np
	}
	return nil
}

// ToolsConfigurable implements Configurable for tool settings.
type ToolsConfigurable struct {
	Enabled []string
}

func (c *ToolsConfigurable) ConfigKey() string { return "tools" }
func (c *ToolsConfigurable) Snapshot() any {
	return map[string]any{
		"enabled": c.Enabled,
	}
}
func (c *ToolsConfigurable) Apply(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("tools: expected map, got %T", v)
	}
	if enabled, ok := m["enabled"]; ok {
		switch val := enabled.(type) {
		case []any:
			c.Enabled = make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					c.Enabled = append(c.Enabled, s)
				}
			}
		case []string:
			c.Enabled = val
		}
	}
	return nil
}
