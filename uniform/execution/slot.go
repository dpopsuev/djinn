// slot.go — YAML-defined agent slots with priority, budget, and model affinity.
//
// Replaces the hardcoded Gear/SupportScheduler system. Operators define
// their own agent roles in YAML. No hardcoded role names in Go code.
//
// Integration: Director fills slots via Troupe Broker.Pick/Spawn.
// Priority drives kill order under budget pressure (lowest dies first).
//
// DJN-TSK-1060
package execution

import "time"

// AgentSlot defines a single agent configuration from YAML.
// The operator writes these — no hardcoded roles in Go.
type AgentSlot struct {
	Role         string     `yaml:"role"`                   // persona name (e.g., "coder", "reviewer")
	Model        string     `yaml:"model,omitempty"`        // model preference (e.g., "opus", "sonnet", "haiku")
	Priority     int        `yaml:"priority"`               // 0=expendable, 100=critical. Kill order: lowest first.
	Capabilities []string   `yaml:"capabilities,omitempty"` // tool capability names from StaffConfig
	Budget       SlotBudget `yaml:"budget,omitempty"`       // per-slot resource limits
	SpawnOn      []string   `yaml:"spawn_on,omitempty"`     // signals that trigger this slot (empty = manual)
}

// SlotBudget defines resource limits for a single agent slot.
type SlotBudget struct {
	MaxTokens   int           `yaml:"max_tokens,omitempty"`   // total token ceiling (in + out); 0 = unlimited
	MaxCost     float64       `yaml:"max_cost,omitempty"`     // cost ceiling in dollars; 0 = unlimited
	MaxDuration time.Duration `yaml:"max_duration,omitempty"` // wall-clock ceiling; 0 = unlimited
}

// SlotTable is an ordered list of agent slots loaded from YAML config.
// Slots are ordered by priority (highest first) for kill-order resolution.
type SlotTable struct {
	Slots    []AgentSlot `yaml:"slots"`
	Capacity int         `yaml:"capacity"` // max concurrent agents (global gate)
}

// DefaultSlotTable returns a minimal single-agent configuration.
func DefaultSlotTable() *SlotTable {
	return &SlotTable{
		Capacity: 1,
		Slots: []AgentSlot{
			{
				Role:     "gensec",
				Model:    "",
				Priority: 100,
				Capabilities: []string{
					"WorkTracking", "SignalBroadcasting", "MemoryRecall",
				},
			},
		},
	}
}

// ForRole returns the slot definition for a role name, or nil if not found.
func (t *SlotTable) ForRole(role string) *AgentSlot {
	for i := range t.Slots {
		if t.Slots[i].Role == role {
			return &t.Slots[i]
		}
	}
	return nil
}

// Roles returns all role names in priority order (highest first).
func (t *SlotTable) Roles() []string {
	roles := make([]string, len(t.Slots))
	for i, s := range t.Slots {
		roles[i] = s.Role
	}
	return roles
}

// KillOrder returns roles ordered by priority ascending (lowest = kill first).
// Roles with priority >= threshold are excluded (protected).
func (t *SlotTable) KillOrder(threshold int) []string {
	var killable []AgentSlot
	for _, s := range t.Slots {
		if s.Priority < threshold {
			killable = append(killable, s)
		}
	}

	// Sort ascending by priority (lowest first).
	for i := 1; i < len(killable); i++ {
		for j := i; j > 0 && killable[j].Priority < killable[j-1].Priority; j-- {
			killable[j], killable[j-1] = killable[j-1], killable[j]
		}
	}

	names := make([]string, len(killable))
	for i, s := range killable {
		names[i] = s.Role
	}
	return names
}

// SlotsForSignal returns slots that should spawn when a signal fires.
func (t *SlotTable) SlotsForSignal(signal string) []AgentSlot {
	var result []AgentSlot
	for _, s := range t.Slots {
		for _, trigger := range s.SpawnOn {
			if trigger == signal {
				result = append(result, s)
				break
			}
		}
	}
	return result
}
