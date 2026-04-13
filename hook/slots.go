// slots.go — converts SlotTable.SpawnOn to unified Hook triggers.
//
// Each AgentSlot.SpawnOn signal becomes an event-phase Hook with
// spawn_slot action. Deleted when SpawnOn field is removed (TSK-1076).
//
// GOL-161, TSK-1073
package hook

import "fmt"

// SlotSpawnConfig represents a slot's spawn trigger for conversion.
type SlotSpawnConfig struct {
	Role    string
	SpawnOn []string
}

// SlotsToHooks converts slot spawn triggers to unified Hooks.
func SlotsToHooks(slots []SlotSpawnConfig) []Hook {
	var hooks []Hook
	for _, slot := range slots {
		for _, sig := range slot.SpawnOn {
			hooks = append(hooks, Hook{
				Name:   fmt.Sprintf("spawn-%s-on-%s", slot.Role, sig),
				On:     PhaseEvent,
				Match:  Matcher{Kind: sig},
				Action: Action{SpawnSlot: slot.Role},
			})
		}
	}
	return hooks
}
