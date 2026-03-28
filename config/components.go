package config

import (
	"errors"
	"fmt"
)

// Agent mode constants.
const (
	ModeAsk   = "ask"
	ModePlan  = "plan"
	ModeAgent = "agent"
	ModeAuto  = "auto"
)

// Sentinel errors for config component parsing.
var (
	ErrUnexpectedType = errors.New("unexpected type")
	ErrUnknownMode    = errors.New("unknown mode")
)

// ModeConfig implements Configurable for the agent mode.
type ModeConfig struct {
	Mode string
}

func (c *ModeConfig) ConfigKey() string { return "mode" }
func (c *ModeConfig) Snapshot() any     { return c.Mode }
func (c *ModeConfig) Apply(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("mode: %w: expected string, got %T", ErrUnexpectedType, v)
	}
	switch s {
	case ModeAsk, ModePlan, ModeAgent, ModeAuto:
		c.Mode = s
		return nil
	default:
		return fmt.Errorf("mode: %w: %q", ErrUnknownMode, s)
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
		return fmt.Errorf("driver: %w: expected map, got %T", ErrUnexpectedType, v)
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
		return fmt.Errorf("session: %w: expected map, got %T", ErrUnexpectedType, v)
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

// SandboxConfigurable implements Configurable for sandbox settings.
type SandboxConfigurable struct {
	Backend string // misbah, bubblewrap, podman, etc.
	Level   string // none, namespace, container, kata
}

func (c *SandboxConfigurable) ConfigKey() string { return "sandbox" }
func (c *SandboxConfigurable) Snapshot() any {
	return map[string]string{
		"backend": c.Backend,
		"level":   c.Level,
	}
}
func (c *SandboxConfigurable) Apply(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("sandbox: %w: expected map, got %T", ErrUnexpectedType, v)
	}
	if b, ok := m["backend"].(string); ok {
		c.Backend = b
	}
	if l, ok := m["level"].(string); ok {
		c.Level = l
	}
	return nil
}

// DebugConfigurable implements Configurable for debug settings.
type DebugConfigurable struct {
	TapFile   string // JSONL capture file path
	LiveDebug string // HTTP debug server address
	Verbose   bool   // show log output on terminal
}

func (c *DebugConfigurable) ConfigKey() string { return "debug" }
func (c *DebugConfigurable) Snapshot() any {
	return map[string]any{
		"tap_file":   c.TapFile,
		"live_debug": c.LiveDebug,
		"verbose":    c.Verbose,
	}
}
func (c *DebugConfigurable) Apply(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("debug: %w: expected map, got %T", ErrUnexpectedType, v)
	}
	if tf, ok := m["tap_file"].(string); ok {
		c.TapFile = tf
	}
	if ld, ok := m["live_debug"].(string); ok {
		c.LiveDebug = ld
	}
	if vb, ok := m["verbose"].(bool); ok {
		c.Verbose = vb
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
		return fmt.Errorf("tools: %w: expected map, got %T", ErrUnexpectedType, v)
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
