// focus_context_integration_test.go — Integration: FocusContext aggregation from multiple providers.
//
// Verifies that FocusContext correctly aggregates context from multiple
// tui.FocusContextProvider implementations and produces prompt-injectable strings.
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

// TestFocusContext_Integration_MultiProvider creates multiple real panels that
// implement tui.FocusContextProvider, collects their FocusContext values, and
// verifies that each produces a valid, structured, prompt-injectable string.
func TestFocusContext_Integration_MultiProvider(t *testing.T) {
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
		Title:   "Implement FocusContext",
		Content: "Wire focus context into prompt injection",
	})
	if err != nil {
		t.Fatalf("graph.Add seg-1: %v", err)
	}
	_, err = graph.Add(artifact.Artifact{
		Kind:    artifact.KindPlanSegment,
		Title:   "Add DebugPanel Provider",
		Content: "DebugPanel implements tui.FocusContextProvider",
	})
	if err != nil {
		t.Fatalf("graph.Add seg-2: %v", err)
	}

	planPanel := widgets.NewPlanPanel(graph)
	planPanel.SetFocus(true)

	// Collect context from both providers.
	providers := []tui.FocusContextProvider{debugPanel, planPanel}

	var combined strings.Builder
	for _, p := range providers {
		fc := p.FocusContext()
		if fc.IsEmpty() {
			t.Errorf("provider %T returned empty FocusContext", p)
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
	tagCount := strings.Count(result, "<focus-context>")
	if tagCount != 2 { //nolint:mnd // exactly 2 providers
		t.Errorf("expected 2 <focus-context> sections, got %d", tagCount)
	}
	closeCount := strings.Count(result, "</focus-context>")
	if closeCount != 2 { //nolint:mnd // exactly 2 providers
		t.Errorf("expected 2 </focus-context> sections, got %d", closeCount)
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
	if !strings.Contains(result, "Implement FocusContext") {
		t.Error("aggregated context should contain plan panel's focused artifact title")
	}

	// Verify metadata from both providers is present.
	if !strings.Contains(result, "component:") {
		t.Error("debug panel context should include component metadata")
	}
	if !strings.Contains(result, "status:") {
		t.Error("plan panel context should include status metadata")
	}
}

// TestFocusContext_Integration_EmptyProviders verifies that FocusContext
// handles providers returning empty context gracefully: no panics,
// and the result is minimal or empty.
func TestFocusContext_Integration_EmptyProviders(t *testing.T) {
	t.Run("no_providers", func(t *testing.T) {
		var providers []tui.FocusContextProvider

		var combined strings.Builder
		for _, p := range providers {
			fc := p.FocusContext()
			combined.WriteString(fc.FormatPrompt())
		}

		if combined.Len() != 0 {
			t.Errorf("no providers should produce empty string, got %q", combined.String())
		}
	})

	t.Run("debug_panel_nil_ring", func(t *testing.T) {
		// DebugPanel with nil ring returns panel-only context (no element).
		panel := widgets.NewDebugPanel(nil)
		fc := panel.FocusContext()

		if fc.PanelID != "debug" {
			t.Errorf("expected PanelID 'debug', got %q", fc.PanelID)
		}
		// ElementID is empty when ring is nil — context reports panel only.
		if fc.ElementID != "" {
			t.Errorf("expected empty ElementID for nil ring, got %q", fc.ElementID)
		}

		// FormatPrompt should still produce output (panel-only context).
		prompt := fc.FormatPrompt()
		if !strings.Contains(prompt, "Panel: debug") {
			t.Error("nil-ring debug panel should still report panel ID")
		}
		if strings.Contains(prompt, "Selected:") {
			t.Error("nil-ring debug panel should not have a selected element")
		}
	})

	t.Run("debug_panel_empty_ring", func(t *testing.T) {
		// DebugPanel with empty ring (no events appended).
		ring := trace.NewRing(16) //nolint:mnd // small ring for test
		panel := widgets.NewDebugPanel(ring)
		fc := panel.FocusContext()

		if fc.PanelID != "debug" {
			t.Errorf("expected PanelID 'debug', got %q", fc.PanelID)
		}
		if fc.ElementID != "" {
			t.Errorf("expected empty ElementID for empty ring, got %q", fc.ElementID)
		}
	})

	t.Run("plan_panel_empty_graph", func(t *testing.T) {
		// PlanPanel with an empty graph (no artifacts).
		registry := artifact.DefaultRegistry()
		graph := artifact.NewGraph("Empty", registry)
		panel := widgets.NewPlanPanel(graph)
		fc := panel.FocusContext()

		if fc.PanelID != "plan" {
			t.Errorf("expected PanelID 'plan', got %q", fc.PanelID)
		}
		if fc.ElementID != "" {
			t.Errorf("expected empty ElementID for empty graph, got %q", fc.ElementID)
		}

		// FormatPrompt for panel-only context.
		prompt := fc.FormatPrompt()
		if !strings.Contains(prompt, "Panel: plan") {
			t.Error("empty-graph plan panel should still report panel ID")
		}
	})

	t.Run("all_empty_providers_aggregated", func(t *testing.T) {
		// Multiple providers, all returning minimal context (panel-only, no element).
		debugEmpty := widgets.NewDebugPanel(nil)
		registry := artifact.DefaultRegistry()
		planEmpty := widgets.NewPlanPanel(artifact.NewGraph("Empty", registry))

		providers := []tui.FocusContextProvider{debugEmpty, planEmpty}

		var combined strings.Builder
		for _, p := range providers {
			fc := p.FocusContext()
			prompt := fc.FormatPrompt()
			combined.WriteString(prompt)
		}

		result := combined.String()
		// Both panels produce panel-only context (PanelID set, no ElementID),
		// so FormatPrompt returns non-empty (it's not IsEmpty since PanelID is set).
		if !strings.Contains(result, "Panel: debug") {
			t.Error("aggregated empty providers should still contain debug panel ID")
		}
		if !strings.Contains(result, "Panel: plan") {
			t.Error("aggregated empty providers should still contain plan panel ID")
		}
		// No Selected lines — no elements focused.
		if strings.Contains(result, "Selected:") {
			t.Error("empty providers should not have any selected elements")
		}
	})
}
