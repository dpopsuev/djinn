// audit.go — ACP capability audit and event routing (GOL-28, TSK-265, TSK-267).
//
// AuditCapabilities documents which ACP features Djinn currently handles.
// RouteACPEvent converts a classified ACP shape into a typed message suitable
// for the TUI event loop.
package acp

import (
	"encoding/json"
	"fmt"
)

// ACPCapability describes a single ACP feature and whether Djinn handles it.
type ACPCapability struct {
	Name        string
	Supported   bool
	Description string
}

// AuditCapabilities returns the known ACP capabilities and their support status.
func AuditCapabilities() []ACPCapability {
	return []ACPCapability{
		{Name: "text_streaming", Supported: true, Description: "Streamed text content from agent"},
		{Name: "thinking", Supported: true, Description: "Agent thinking/reasoning stream"},
		{Name: "tool_use", Supported: true, Description: "Agent tool call requests and results"},
		{Name: "plan_update", Supported: true, Description: "Structured plan steps via ShapeClassifier"},
		{Name: "diff_update", Supported: true, Description: "File diff/change events via ShapeClassifier"},
		{Name: "state_change", Supported: true, Description: "Key-value state transitions via ShapeClassifier"},
		{Name: "capability_list", Supported: true, Description: "Agent capability enumeration via ShapeClassifier"},
		{Name: "billing_info", Supported: false, Description: "Token usage and cost reporting (TSK-266, future)"},
		{Name: "project_index", Supported: false, Description: "Project file indexing events (TSK-268, future)"},
	}
}

// --- Routed message types (driver-layer, no TUI dependency). ---

// ACPPlanUpdateMsg carries plan steps extracted from an ACP event.
type ACPPlanUpdateMsg struct{ Steps []string }

// ACPDiffUpdateMsg carries file paths from a diff ACP event.
type ACPDiffUpdateMsg struct{ Files []string }

// ACPStateChangeMsg carries a key-value state change.
type ACPStateChangeMsg struct{ Key, Value string }

// ACPOutputMsg carries formatted output for unknown/fallback shapes.
type ACPOutputMsg struct{ Line string }

// RouteACPEvent converts a classified ACP shape into a typed message.
//
// The returned value is one of:
//   - ACPPlanUpdateMsg  (ShapePlanUpdate)
//   - ACPDiffUpdateMsg  (ShapeDiffUpdate)
//   - ACPStateChangeMsg (ShapeStateChange)
//   - ACPOutputMsg      (ShapeUnknown or unhandled)
//
// Callers in the TUI layer map these to the appropriate tea.Msg types.
func RouteACPEvent(shape ShapeKind, data map[string]any) any {
	switch shape {
	case ShapePlanUpdate:
		return ACPPlanUpdateMsg{Steps: extractStringSlice(data, "steps", "plan")}

	case ShapeDiffUpdate:
		return ACPDiffUpdateMsg{Files: extractFileList(data)}

	case ShapeStateChange:
		key, _ := data["key"].(string)
		value := fmt.Sprintf("%v", data["value"])
		return ACPStateChangeMsg{Key: key, Value: value}

	default:
		// Format unknown shapes as indented JSON for display.
		raw, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return ACPOutputMsg{Line: fmt.Sprintf("[unknown ACP event: %v]", data)}
		}
		return ACPOutputMsg{Line: string(raw)}
	}
}

// extractStringSlice extracts a string slice from the first matching key in data.
func extractStringSlice(data map[string]any, keys ...string) []string {
	for _, key := range keys {
		v, ok := data[key]
		if !ok {
			continue
		}
		arr, isArr := v.([]any)
		if !isArr {
			continue
		}
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	}
	return nil
}

// extractFileList extracts file paths from "diff" or "changes" arrays.
func extractFileList(data map[string]any) []string {
	// Try "diff" as a string slice first.
	if files := extractStringSlice(data, "diff"); len(files) > 0 {
		return files
	}

	// Try "changes" with nested "file" keys.
	v, ok := data["changes"]
	if !ok {
		return nil
	}
	arr, isArr := v.([]any)
	if !isArr {
		return nil
	}
	var files []string
	for _, item := range arr {
		m, isMap := item.(map[string]any)
		if !isMap {
			continue
		}
		if f, ok := m["file"].(string); ok {
			files = append(files, f)
		}
	}
	return files
}
