// render_integration_test.go — Integration test: RenderTool → handler intercept → RenderPanel (TSK-506).
//
// Tests the full render pipeline across component boundaries WITHOUT
// Model or ScriptedDriver. Pure function composition: Execute → parse → render.
package tui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/tools/builtin"
)

// simulateRenderPipeline runs the full render pipeline:
// RenderTool.Execute → parse output (handler intercept logic) → RenderPanel.
func simulateRenderPipeline(t *testing.T, input json.RawMessage) string {
	t.Helper()

	tool := &builtin.RenderTool{}
	output, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("RenderTool.Execute failed: %v", err)
	}

	// Simulate handler intercept logic (tui/handler.go:52-63).
	var render struct {
		Type  string `json:"type"`
		Title string `json:"title"`
		Data  string `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &render); err != nil {
		t.Fatalf("handler intercept unmarshal failed: %v", err)
	}
	if render.Type == "" {
		t.Fatal("handler intercept: empty type")
	}

	// Construct RenderPanelMsg and render.
	msg := RenderPanelMsg{
		Type:  render.Type,
		Title: render.Title,
		Data:  render.Data,
	}
	return RenderPanel(msg, 80) //nolint:mnd // standard test width
}

// --- Diagram ---

func TestRenderPipeline_Diagram(t *testing.T) {
	input := mustJSON(t, map[string]any{
		"type":  "diagram",
		"title": "Plan Dependencies",
		"data": mustJSONStr(t, map[string]any{
			"nodes": []map[string]string{
				{"id": "gol-58", "label": "Hub Mediators"},
				{"id": "gol-57", "label": "Planner Port"},
			},
			"edges": []map[string]string{
				{"from": "gol-58", "to": "gol-57"},
			},
		}),
	})

	result := simulateRenderPipeline(t, input)

	if !strings.Contains(result, "Hub Mediators") {
		t.Error("diagram should contain node label 'Hub Mediators'")
	}
	if !strings.Contains(result, "Planner Port") {
		t.Error("diagram should contain node label 'Planner Port'")
	}
	if !strings.Contains(result, "→") {
		t.Error("diagram should contain arrow connector")
	}
	if !strings.Contains(result, "Plan Dependencies") {
		t.Error("diagram should contain title")
	}
}

// --- Progress ---

func TestRenderPipeline_Progress(t *testing.T) {
	input := mustJSON(t, map[string]any{
		"type":  "progress",
		"title": "Campaign Progress",
		"data": mustJSONStr(t, map[string]any{
			"done":  3,
			"total": 5,
			"items": []string{"Hub Mediators", "Planner Port", "Shell Maturity", "Locus Port", "TUI Decomposition"},
		}),
	})

	result := simulateRenderPipeline(t, input)

	if !strings.Contains(result, "█") {
		t.Error("progress should contain filled bar characters")
	}
	if !strings.Contains(result, "60%") {
		t.Error("progress should show 60% completion")
	}
	if !strings.Contains(result, "☑") {
		t.Error("progress should contain checked items")
	}
	if !strings.Contains(result, "☐") {
		t.Error("progress should contain unchecked items")
	}
}

// --- Tree ---

func TestRenderPipeline_Tree(t *testing.T) {
	input := mustJSON(t, map[string]any{
		"type":  "tree",
		"title": "Plan Hierarchy",
		"data": mustJSONStr(t, map[string]any{
			"root": map[string]any{
				"label": "CMP-9 Enhanced Planning",
				"children": []map[string]any{
					{"label": "GOL-59 Artifact Primitive"},
					{"label": "GOL-60 Render Awareness"},
					{"label": "GOL-61 Plan TUI"},
				},
			},
		}),
	})

	result := simulateRenderPipeline(t, input)

	if !strings.Contains(result, "CMP-9") {
		t.Error("tree should contain root label")
	}
	if !strings.Contains(result, "├") || !strings.Contains(result, "└") {
		t.Error("tree should contain connector characters (├ └)")
	}
	if !strings.Contains(result, "GOL-59") {
		t.Error("tree should contain child labels")
	}
}

// --- Timeline ---

func TestRenderPipeline_Timeline(t *testing.T) {
	input := mustJSON(t, map[string]any{
		"type":  "timeline",
		"title": "Execution Order",
		"data": mustJSONStr(t, map[string]any{
			"events": []map[string]string{
				{"label": "Hub Mediators", "state": "done"},
				{"label": "Planner Port", "state": "done"},
				{"label": "Shell Maturity", "state": "active"},
				{"label": "TUI Decomposition", "state": "pending"},
			},
		}),
	})

	result := simulateRenderPipeline(t, input)

	if !strings.Contains(result, "Hub Mediators") {
		t.Error("timeline should contain event labels")
	}
	if !strings.Contains(result, "●") {
		t.Error("timeline should contain filled dot for done/active events")
	}
	if !strings.Contains(result, "○") {
		t.Error("timeline should contain hollow dot for pending events")
	}
}

// --- Error: invalid type ---

func TestRenderPipeline_InvalidType(t *testing.T) {
	tool := &builtin.RenderTool{}
	input := mustJSON(t, map[string]any{
		"type":  "alien",
		"title": "Bad",
		"data":  "{}",
	})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for invalid render type")
	}
}

// --- Helpers ---

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustJSONStr(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
