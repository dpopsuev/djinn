// sight_integration_test.go — Integration: CellSight aggregation from multiple providers.
//
// Verifies that CellSight correctly aggregates context from multiple
// tui.Sighted implementations and produces prompt-injectable strings.
// Uses real panel types (DebugPanel, PlanPanel) as providers.
package tui_test

import (
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/trace"
	"github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/tui/widgets"
)

// TestCellSight_Integration_MultiProvider creates multiple real panels that
// implement tui.Sighted, collects their CellSight values, and
// verifies that each produces a valid, structured, prompt-injectable string.
func TestCellSight_Integration_MultiProvider(t *testing.T) {
	// --- Provider 1: DebugPanel with trace events ---
	ring := trace.NewRing(64) //nolint:mnd // small ring for test
	ring.Append(trace.TraceEvent{
		Component: trace.ComponentMCP,
		Action:    "call",
		Server:    "scribe",
		Tool:      "artifact",
		Detail:    "get DJN-SPC-2026-001",
		Latency:   42 * time.Millisecond, //nolint:mnd // test latency
	})
	ring.Append(trace.TraceEvent{
		Component: trace.ComponentTool,
		Action:    "result",
		Server:    "locus",
		Tool:      "analysis",
		Detail:    "code health report",
		Latency:   150 * time.Millisecond, //nolint:mnd // test latency
	})
	debugPanel := widgets.NewDebugPanel(ring)
	debugPanel.SetFocus(true)

	// --- Provider 2: PlanPanel with artifacts ---
	registry := artifact.DefaultRegistry()
	graph := artifact.NewGraph("Test Plan", registry)

	_, err := graph.Add(artifact.Artifact{
		Kind:    artifact.KindPlanSegment,
		Title:   "Implement CellSight",
		Content: "Wire cell sight into prompt injection",
	})
	if err != nil {
		t.Fatalf("graph.Add seg-1: %v", err)
	}
	_, err = graph.Add(artifact.Artifact{
		Kind:    artifact.KindPlanSegment,
		Title:   "Add DebugPanel Provider",
		Content: "DebugPanel implements tui.Sighted",
	})
	if err != nil {
		t.Fatalf("graph.Add seg-2: %v", err)
	}

	planPanel := widgets.NewPlanPanel(graph)
	planPanel.SetFocus(true)

	// Collect context from both providers.
	providers := []tui.Sighted{debugPanel, planPanel}

	var combined strings.Builder
	for _, p := range providers {
		if !p.SightGate() {
			continue
		}
		fc := p.CellSight()
		if fc.IsEmpty() {
			t.Errorf("provider %T returned empty CellSight", p)
			continue
		}
		prompt := fc.FormatPrompt()
		if prompt == "" {
			t.Errorf("provider %T returned empty FormatPrompt()", p)
			continue
		}
		combined.WriteString(prompt)
		combined.WriteString("\n\n")
	}

	result := combined.String()
	if result == "" {
		t.Fatal("aggregated context from multiple providers should not be empty")
	}

	// Verify structural markers present from both providers.
	tagCount := strings.Count(result, "<cell-sight>")
	if tagCount != 2 { //nolint:mnd // exactly 2 providers
		t.Errorf("expected 2 <cell-sight> sections, got %d", tagCount)
	}
	closeCount := strings.Count(result, "</cell-sight>")
	if closeCount != 2 { //nolint:mnd // exactly 2 providers
		t.Errorf("expected 2 </cell-sight> sections, got %d", closeCount)
	}

	// Verify DebugPanel context appears (newest event = "code health report").
	if !strings.Contains(result, "Panel: debug") {
		t.Error("aggregated context should contain debug panel")
	}
	if !strings.Contains(result, "code health report") {
		t.Error("aggregated context should contain debug panel's focused event detail")
	}

	// Verify PlanPanel context appears (cursor at seg-1 in overview).
	if !strings.Contains(result, "Panel: plan") {
		t.Error("aggregated context should contain plan panel")
	}
	if !strings.Contains(result, "Implement CellSight") {
		t.Error("aggregated context should contain plan panel's focused artifact title")
	}

	// Verify non-sensitive fields from both providers are present.
	if !strings.Contains(result, "component:") {
		t.Error("debug panel context should include component field")
	}
	if !strings.Contains(result, "status:") {
		t.Error("plan panel context should include status field")
	}

	// Verify sensitive fields (latency) are hidden.
	if strings.Contains(result, "latency: 150ms") {
		t.Error("sensitive latency field should NOT appear in prompt")
	}
	if !strings.Contains(result, "fields hidden") {
		t.Error("hidden hint should appear when sensitive fields exist")
	}
}

// TestCellSight_Integration_EmptyProviders verifies that CellSight
// handles providers returning empty context gracefully: no panics,
// and the result is minimal or empty.
func TestCellSight_Integration_EmptyProviders(t *testing.T) {
	t.Run("no_providers", func(t *testing.T) {
		var providers []tui.Sighted

		var combined strings.Builder
		for _, p := range providers {
			fc := p.CellSight()
			combined.WriteString(fc.FormatPrompt())
		}

		if combined.Len() != 0 {
			t.Errorf("no providers should produce empty string, got %q", combined.String())
		}
	})

	t.Run("debug_panel_nil_ring", func(t *testing.T) {
		// DebugPanel with nil ring returns panel-only context (no cell).
		panel := widgets.NewDebugPanel(nil)
		fc := panel.CellSight()

		if fc.PanelID != "debug" {
			t.Errorf("expected PanelID 'debug', got %q", fc.PanelID)
		}
		// CellID is empty when ring is nil — context reports panel only.
		if fc.CellID != "" {
			t.Errorf("expected empty CellID for nil ring, got %q", fc.CellID)
		}

		// FormatPrompt should still produce output (panel-only context).
		prompt := fc.FormatPrompt()
		if !strings.Contains(prompt, "Panel: debug") {
			t.Error("nil-ring debug panel should still report panel ID")
		}
		if strings.Contains(prompt, "Selected:") {
			t.Error("nil-ring debug panel should not have a selected cell")
		}
	})

	t.Run("debug_panel_empty_ring", func(t *testing.T) {
		// DebugPanel with empty ring (no events appended).
		ring := trace.NewRing(16) //nolint:mnd // small ring for test
		panel := widgets.NewDebugPanel(ring)
		fc := panel.CellSight()

		if fc.PanelID != "debug" {
			t.Errorf("expected PanelID 'debug', got %q", fc.PanelID)
		}
		if fc.CellID != "" {
			t.Errorf("expected empty CellID for empty ring, got %q", fc.CellID)
		}
	})

	t.Run("plan_panel_empty_graph", func(t *testing.T) {
		// PlanPanel with an empty graph (no artifacts).
		registry := artifact.DefaultRegistry()
		graph := artifact.NewGraph("Empty", registry)
		panel := widgets.NewPlanPanel(graph)
		fc := panel.CellSight()

		if fc.PanelID != "plan" {
			t.Errorf("expected PanelID 'plan', got %q", fc.PanelID)
		}
		if fc.CellID != "" {
			t.Errorf("expected empty CellID for empty graph, got %q", fc.CellID)
		}

		// FormatPrompt for panel-only context.
		prompt := fc.FormatPrompt()
		if !strings.Contains(prompt, "Panel: plan") {
			t.Error("empty-graph plan panel should still report panel ID")
		}
	})

	t.Run("all_empty_providers_aggregated", func(t *testing.T) {
		// Multiple providers, all returning minimal context (panel-only, no cell).
		debugEmpty := widgets.NewDebugPanel(nil)
		registry := artifact.DefaultRegistry()
		planEmpty := widgets.NewPlanPanel(artifact.NewGraph("Empty", registry))

		providers := []tui.Sighted{debugEmpty, planEmpty}

		var combined strings.Builder
		for _, p := range providers {
			fc := p.CellSight()
			prompt := fc.FormatPrompt()
			combined.WriteString(prompt)
		}

		result := combined.String()
		// Both panels produce panel-only context (PanelID set, no CellID),
		// so FormatPrompt returns non-empty (it's not IsEmpty since PanelID is set).
		if !strings.Contains(result, "Panel: debug") {
			t.Error("aggregated empty providers should still contain debug panel ID")
		}
		if !strings.Contains(result, "Panel: plan") {
			t.Error("aggregated empty providers should still contain plan panel ID")
		}
		// No Selected lines — no cells focused.
		if strings.Contains(result, "Selected:") {
			t.Error("empty providers should not have any selected cells")
		}
	})
}

// TestSightGate_False_NoInjection verifies that when SightGate returns false,
// the panel's CellSight is not injected into the prompt.
func TestSightGate_False_NoInjection(t *testing.T) {
	// Both real panels return SightGate() == true by default.
	// Verify the gate check logic works by simulating the model.go pattern.
	ring := trace.NewRing(16) //nolint:mnd // small ring for test
	ring.Append(trace.TraceEvent{
		Component: trace.ComponentMCP,
		Action:    "call",
		Detail:    "should appear",
	})
	panel := widgets.NewDebugPanel(ring)

	// SightGate is true — injection should happen.
	if !panel.SightGate() {
		t.Fatal("DebugPanel SightGate should be true by default")
	}

	// Simulate the gate check from model.go.
	var prompt string
	provider := tui.Sighted(panel)
	if provider.SightGate() {
		if fc := provider.CellSight(); !fc.IsEmpty() {
			prompt = fc.FormatPrompt()
		}
	}
	if prompt == "" {
		t.Error("with SightGate() == true, prompt should contain cell sight")
	}

	// Now test with a mock that returns SightGate() == false.
	gated := &gatedPanel{sight: panel.CellSight(), gate: false}
	var gatedPrompt string
	if gated.SightGate() {
		if fc := gated.CellSight(); !fc.IsEmpty() {
			gatedPrompt = fc.FormatPrompt()
		}
	}
	if gatedPrompt != "" {
		t.Error("with SightGate() == false, prompt should be empty")
	}
}

// gatedPanel is a test helper that wraps a CellSight with a configurable gate.
type gatedPanel struct {
	sight tui.CellSight
	gate  bool
}

func (g *gatedPanel) CellSight() tui.CellSight { return g.sight }
func (g *gatedPanel) SightGate() bool          { return g.gate }

// Compile-time check.
var _ tui.Sighted = (*gatedPanel)(nil)
