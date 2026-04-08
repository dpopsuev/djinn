// sight_manager.go — Operator-controlled agent visibility (GOL-62).
//
// SightManager tracks per-panel gate state and per-field reveal overrides.
// The operator uses :sight commands to toggle what the agent can see.
//
// Gates default to open (true). Fields default to their panel-declared
// Sensitive flag; Reveal/Hide override on a per-field basis.
package tui

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/dpopsuev/djinn/telemetry"
)

// SightManager tracks gate state and field overrides per panel.
// Thread-safe: all methods are safe for concurrent use.
// All mutations are logged for audit trail (OWASP A09).
type SightManager struct {
	mu      sync.RWMutex
	gates   map[string]bool // panel ID -> gate on/off
	reveals map[string]bool // "panel.field" -> revealed (true) or hidden (false)
	log     *slog.Logger
}

// NewSightManager creates a SightManager with default-open gates.
func NewSightManager(log *slog.Logger) *SightManager {
	if log == nil {
		log = telemetry.Nop()
	}
	return &SightManager{
		gates:   make(map[string]bool),
		reveals: make(map[string]bool),
		log:     log,
	}
}

// SetGate enables or disables the visibility gate for a panel.
// When off, the panel's CellSight is not injected into the agent prompt.
func (m *SightManager) SetGate(panelID string, on bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gates[panelID] = on

	// Yellow: operator visibility decision is auditable
	action := "gate_off"
	if on {
		action = "gate_on"
	}
	m.log.InfoContext(context.Background(), "sight gate changed",
		slog.String(telemetry.KeyComponent, "sight"),
		slog.String(telemetry.KeyAction, action),
		slog.String(telemetry.KeyPanel, panelID),
	)
}

// IsGateOpen returns whether a panel's gate is open.
// Panels not explicitly set default to open (true).
func (m *SightManager) IsGateOpen(panelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.gates[panelID]; ok {
		return v
	}
	return true // default open
}

// Reveal overrides a field's Sensitive flag to false, making it visible to the agent.
func (m *SightManager) Reveal(panelID, field string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reveals[panelID+"."+field] = true

	// Yellow: operator reveals sensitive field — audit trail
	m.log.InfoContext(context.Background(), "sight field revealed",
		slog.String(telemetry.KeyComponent, "sight"),
		slog.String(telemetry.KeyAction, "reveal"),
		slog.String(telemetry.KeyPanel, panelID),
		slog.String(telemetry.KeyField, field),
	)
}

// Hide restores a field's Sensitive flag, making it hidden from the agent.
func (m *SightManager) Hide(panelID, field string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reveals[panelID+"."+field] = false

	// Yellow: operator hides field — audit trail
	m.log.InfoContext(context.Background(), "sight field hidden",
		slog.String(telemetry.KeyComponent, "sight"),
		slog.String(telemetry.KeyAction, "hide"),
		slog.String(telemetry.KeyPanel, panelID),
		slog.String(telemetry.KeyField, field),
	)
}

// IsRevealed returns whether a field has been explicitly revealed.
// Returns false if the field has no override (panel's default applies).
func (m *SightManager) IsRevealed(panelID, field string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reveals[panelID+"."+field]
}

// Status returns a human-readable description of the current sight state.
func (m *SightManager) Status() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder
	b.WriteString("sight:\n")

	if len(m.gates) == 0 && len(m.reveals) == 0 {
		b.WriteString("  all panels open (defaults)\n")
		b.WriteString("  no field overrides")
		return b.String()
	}

	// Gates section.
	if len(m.gates) > 0 {
		b.WriteString("  gates:\n")
		panels := make([]string, 0, len(m.gates))
		for p := range m.gates {
			panels = append(panels, p)
		}
		sort.Strings(panels)
		for _, p := range panels {
			state := "on"
			if !m.gates[p] {
				state = "off"
			}
			fmt.Fprintf(&b, "    %s: %s\n", p, state)
		}
	}

	// Field overrides section.
	if len(m.reveals) > 0 {
		b.WriteString("  fields:\n")
		keys := make([]string, 0, len(m.reveals))
		for k := range m.reveals {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			state := "hidden"
			if m.reveals[k] {
				state = "revealed"
			}
			fmt.Fprintf(&b, "    %s: %s\n", k, state)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// ApplyCellSight returns a copy of the CellSight with field sensitivity
// adjusted according to SightManager overrides. Fields that have been
// Reveal'd have their Sensitive flag cleared; fields that have been
// Hide'd have their Sensitive flag set.
func (m *SightManager) ApplyCellSight(cs CellSight) CellSight {
	if len(cs.Fields) == 0 {
		return cs
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Copy fields slice to avoid mutating the original.
	fields := make([]SightField, len(cs.Fields))
	copy(fields, cs.Fields)

	for i := range fields {
		key := cs.PanelID + "." + fields[i].Key
		if override, ok := m.reveals[key]; ok {
			if override {
				fields[i].Sensitive = false
			} else {
				fields[i].Sensitive = true
			}
		}
	}

	return CellSight{
		PanelID:   cs.PanelID,
		CellID:    cs.CellID,
		CellTitle: cs.CellTitle,
		Kind:      cs.Kind,
		Fields:    fields,
	}
}
