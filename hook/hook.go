// hook.go — Hook, Matcher, Action types + YAML parsing.
//
// GOL-161, TSK-1068
package hook

import (
	"errors"
	"fmt"

	"github.com/dpopsuev/troupe/signal"
	"gopkg.in/yaml.v3"
)

// Validation errors.
var (
	ErrNameRequired  = errors.New("hook: name is required")
	ErrDuplicateName = errors.New("hook: duplicate name")
	ErrUnknownPhase  = errors.New("hook: unknown phase")
	ErrNoAction      = errors.New("hook: at least one action is required")
)

// Phase classifies when a hook fires relative to execution.
type Phase string

const (
	PhasePreToolUse  Phase = "pre_tool_use"  // synchronous gate, can deny
	PhasePostToolUse Phase = "post_tool_use" // synchronous recorder, observe only
	PhaseEvent       Phase = "event"         // async, reacts to EventLog events
)

// Hook is a single declarative hook loaded from YAML.
// Zero LLM involvement — purely deterministic matching and actions.
type Hook struct {
	Name   string  `yaml:"name"`
	On     Phase   `yaml:"on"`
	Match  Matcher `yaml:"match,omitempty"`
	Action Action  `yaml:"action"`
	Scope  string  `yaml:"scope,omitempty"` // empty = global, otherwise prefix-matches Event.Source
}

// Matcher defines what events/tool calls this hook applies to.
type Matcher struct {
	Tool string `yaml:"tool,omitempty"` // tool name or "*" for wildcard
	Kind string `yaml:"kind,omitempty"` // event kind (for Phase "event")
}

// Action defines what happens when a hook fires.
type Action struct {
	Deny      string `yaml:"deny,omitempty"`       // non-empty = deny tool call with this reason
	Emit      string `yaml:"emit,omitempty"`       // emit a new event with this Kind
	SpawnSlot string `yaml:"spawn_slot,omitempty"` // spawn agent slot by role name
	Shell     string `yaml:"shell,omitempty"`      // run shell command (exit 2 = deny)
}

// HooksConfig is the top-level YAML structure for hook definitions.
type HooksConfig struct {
	Hooks []Hook `yaml:"hooks"`
}

// ParseHooks parses hook configuration from YAML bytes.
func ParseHooks(data []byte) (HooksConfig, error) {
	var cfg HooksConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse hooks: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Validate checks that all hooks have valid phases and non-empty names.
func (c *HooksConfig) Validate() error {
	seen := make(map[string]bool)
	for i := range c.Hooks {
		h := &c.Hooks[i]
		if h.Name == "" {
			return fmt.Errorf("%w: index %d", ErrNameRequired, i)
		}
		if seen[h.Name] {
			return fmt.Errorf("%w: %q", ErrDuplicateName, h.Name)
		}
		seen[h.Name] = true

		switch h.On {
		case PhasePreToolUse, PhasePostToolUse, PhaseEvent:
			// valid
		default:
			return fmt.Errorf("%w: %q on hook %q", ErrUnknownPhase, h.On, h.Name)
		}

		if h.Action == (Action{}) {
			return fmt.Errorf("%w: hook %q", ErrNoAction, h.Name)
		}
	}
	return nil
}

// MatchesTool returns true if the matcher applies to the given tool name.
func (m Matcher) MatchesTool(tool string) bool {
	if m.Tool == "" || m.Tool == "*" {
		return true
	}
	return m.Tool == tool
}

// MatchesEvent returns true if the matcher applies to the given event.
func (m Matcher) MatchesEvent(e signal.Event) bool {
	if m.Kind != "" && m.Kind != e.Kind {
		return false
	}
	return true
}

// MatchesScope returns true if the hook applies to the given event source.
// Empty scope = global (matches everything).
func (h Hook) MatchesScope(source string) bool {
	if h.Scope == "" {
		return true
	}
	if len(source) < len(h.Scope) {
		return source == h.Scope
	}
	return source[:len(h.Scope)] == h.Scope
}
