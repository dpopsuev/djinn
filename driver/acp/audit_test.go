package acp

import (
	"testing"
)

func TestAuditCapabilities_Count(t *testing.T) {
	caps := AuditCapabilities()
	if len(caps) != 9 {
		t.Fatalf("AuditCapabilities() returned %d items, want 9", len(caps))
	}
}

func TestAuditCapabilities_SupportedList(t *testing.T) {
	caps := AuditCapabilities()
	supported := make(map[string]bool)
	for _, c := range caps {
		supported[c.Name] = c.Supported
	}

	wantSupported := []string{
		"text_streaming", "thinking", "tool_use",
		"plan_update", "diff_update", "state_change", "capability_list",
	}
	for _, name := range wantSupported {
		if !supported[name] {
			t.Errorf("capability %q should be supported", name)
		}
	}

	wantUnsupported := []string{"billing_info", "project_index"}
	for _, name := range wantUnsupported {
		if supported[name] {
			t.Errorf("capability %q should NOT be supported (future)", name)
		}
	}
}

func TestAuditCapabilities_DescriptionsNonEmpty(t *testing.T) {
	for _, c := range AuditCapabilities() {
		if c.Description == "" {
			t.Errorf("capability %q has empty description", c.Name)
		}
	}
}

func TestRouteACPEvent_PlanUpdate(t *testing.T) {
	data := map[string]any{
		"steps": []any{"step 1", "step 2", "step 3"},
	}
	msg := RouteACPEvent(ShapePlanUpdate, data)

	plan, ok := msg.(ACPPlanUpdateMsg)
	if !ok {
		t.Fatalf("expected ACPPlanUpdateMsg, got %T", msg)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("Steps len = %d, want 3", len(plan.Steps))
	}
	if plan.Steps[0] != "step 1" {
		t.Errorf("Steps[0] = %q, want %q", plan.Steps[0], "step 1")
	}
}

func TestRouteACPEvent_PlanUpdateViaPlanKey(t *testing.T) {
	data := map[string]any{
		"plan": []any{"phase A", "phase B"},
	}
	msg := RouteACPEvent(ShapePlanUpdate, data)

	plan, ok := msg.(ACPPlanUpdateMsg)
	if !ok {
		t.Fatalf("expected ACPPlanUpdateMsg, got %T", msg)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(plan.Steps))
	}
}

func TestRouteACPEvent_DiffUpdateViaArray(t *testing.T) {
	data := map[string]any{
		"diff": []any{"file1.go", "file2.go"},
	}
	msg := RouteACPEvent(ShapeDiffUpdate, data)

	diff, ok := msg.(ACPDiffUpdateMsg)
	if !ok {
		t.Fatalf("expected ACPDiffUpdateMsg, got %T", msg)
	}
	if len(diff.Files) != 2 {
		t.Fatalf("Files len = %d, want 2", len(diff.Files))
	}
	if diff.Files[0] != "file1.go" {
		t.Errorf("Files[0] = %q, want %q", diff.Files[0], "file1.go")
	}
}

func TestRouteACPEvent_DiffUpdateViaChanges(t *testing.T) {
	data := map[string]any{
		"changes": []any{
			map[string]any{"file": "main.go", "action": "modify"},
			map[string]any{"file": "test.go", "action": "create"},
		},
	}
	msg := RouteACPEvent(ShapeDiffUpdate, data)

	diff, ok := msg.(ACPDiffUpdateMsg)
	if !ok {
		t.Fatalf("expected ACPDiffUpdateMsg, got %T", msg)
	}
	if len(diff.Files) != 2 {
		t.Fatalf("Files len = %d, want 2", len(diff.Files))
	}
	if diff.Files[1] != "test.go" {
		t.Errorf("Files[1] = %q, want %q", diff.Files[1], "test.go")
	}
}

func TestRouteACPEvent_StateChange(t *testing.T) {
	data := map[string]any{
		"key":   "mode",
		"value": "agent",
	}
	msg := RouteACPEvent(ShapeStateChange, data)

	sc, ok := msg.(ACPStateChangeMsg)
	if !ok {
		t.Fatalf("expected ACPStateChangeMsg, got %T", msg)
	}
	if sc.Key != "mode" {
		t.Errorf("Key = %q, want %q", sc.Key, "mode")
	}
	if sc.Value != "agent" {
		t.Errorf("Value = %q, want %q", sc.Value, "agent")
	}
}

func TestRouteACPEvent_StateChangeNonStringValue(t *testing.T) {
	data := map[string]any{
		"key":   "count",
		"value": 42,
	}
	msg := RouteACPEvent(ShapeStateChange, data)

	sc, ok := msg.(ACPStateChangeMsg)
	if !ok {
		t.Fatalf("expected ACPStateChangeMsg, got %T", msg)
	}
	if sc.Value != "42" {
		t.Errorf("Value = %q, want %q", sc.Value, "42")
	}
}

func TestRouteACPEvent_Unknown(t *testing.T) {
	data := map[string]any{
		"random": "payload",
	}
	msg := RouteACPEvent(ShapeUnknown, data)

	out, ok := msg.(ACPOutputMsg)
	if !ok {
		t.Fatalf("expected ACPOutputMsg, got %T", msg)
	}
	if out.Line == "" {
		t.Error("ACPOutputMsg.Line should not be empty for unknown shape")
	}
}

func TestRouteACPEvent_TextStream(t *testing.T) {
	// ShapeTextStream is not explicitly handled — falls to default.
	data := map[string]any{"content": "hello"}
	msg := RouteACPEvent(ShapeTextStream, data)

	_, ok := msg.(ACPOutputMsg)
	if !ok {
		t.Fatalf("expected ACPOutputMsg for unhandled shape, got %T", msg)
	}
}

func TestRouteACPEvent_EmptyPlanSteps(t *testing.T) {
	data := map[string]any{
		"steps": []any{},
	}
	msg := RouteACPEvent(ShapePlanUpdate, data)

	plan, ok := msg.(ACPPlanUpdateMsg)
	if !ok {
		t.Fatalf("expected ACPPlanUpdateMsg, got %T", msg)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("Steps should be empty, got %d", len(plan.Steps))
	}
}
