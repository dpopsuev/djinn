package tui

import (
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// RED: Invalid/empty data — tests validate error handling
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTable_InvalidJSON(t *testing.T) {
	result := renderTable("not json", 80)
	if !strings.Contains(result, "invalid table data") {
		t.Fatalf("result = %q, want invalid table data", result)
	}
}

func TestRenderTree_InvalidJSON(t *testing.T) {
	result := renderTree("not json", 80)
	if !strings.Contains(result, "invalid tree data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderProgress_InvalidJSON(t *testing.T) {
	result := renderProgress("not json", 80)
	if !strings.Contains(result, "invalid progress data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderChart_InvalidJSON(t *testing.T) {
	result := renderChart("not json", 80)
	if !strings.Contains(result, "invalid chart data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderPanel_UnknownType(t *testing.T) {
	msg := RenderPanelMsg{Type: "unknown_type", Title: "Test", Data: "{}"}
	result := RenderPanel(msg, 80)
	if !strings.Contains(result, "unknown render type") {
		t.Fatalf("result = %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GREEN: Happy path — correct rendering
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTable_BasicTable(t *testing.T) {
	data := `{"columns":["Name","Score"],"rows":[["Alice","95"],["Bob","87"]]}`
	result := renderTable(data, 80)

	if !strings.Contains(result, "Name") {
		t.Fatal("missing column header 'Name'")
	}
	if !strings.Contains(result, "Score") {
		t.Fatal("missing column header 'Score'")
	}
	if !strings.Contains(result, "Alice") {
		t.Fatal("missing row value 'Alice'")
	}
	if !strings.Contains(result, "87") {
		t.Fatal("missing row value '87'")
	}
	// Should have separator line.
	if !strings.Contains(result, "─") {
		t.Fatal("missing separator")
	}
}

func TestRenderTable_ColumnWidthAdapts(t *testing.T) {
	data := `{"columns":["X","LongColumnName"],"rows":[["1","2"]]}`
	result := renderTable(data, 80)
	lines := strings.Split(result, "\n")
	// Header and separator should be at least as wide as "LongColumnName".
	for _, line := range lines {
		if strings.Contains(line, "LongColumnName") {
			return // found it
		}
	}
	t.Fatal("column header not rendered")
}

func TestRenderTree_SimpleHierarchy(t *testing.T) {
	data := `{"root":{"label":"root","children":[{"label":"child1"},{"label":"child2"}]}}`
	result := renderTree(data, 80)

	if !strings.Contains(result, "root") {
		t.Fatal("missing root")
	}
	if !strings.Contains(result, "├── ") {
		t.Fatal("missing ├── connector")
	}
	if !strings.Contains(result, "└── ") {
		t.Fatal("missing └── connector")
	}
	if !strings.Contains(result, "child1") || !strings.Contains(result, "child2") {
		t.Fatal("missing children")
	}
}

func TestRenderTree_DeepNesting(t *testing.T) {
	data := `{"root":{"label":"a","children":[{"label":"b","children":[{"label":"c","children":[{"label":"d"}]}]}]}}`
	result := renderTree(data, 80)

	if !strings.Contains(result, "a") || !strings.Contains(result, "d") {
		t.Fatal("missing nested nodes")
	}
	// Should have indentation.
	lines := strings.Split(result, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected 4+ lines, got %d", len(lines))
	}
}

func TestRenderProgress_Percentage(t *testing.T) {
	data := `{"done":3,"total":4}`
	result := renderProgress(data, 80)

	if !strings.Contains(result, "75") {
		t.Fatalf("result = %q, want 75%%", result)
	}
	if !strings.Contains(result, "█") {
		t.Fatal("missing filled blocks")
	}
	if !strings.Contains(result, "░") {
		t.Fatal("missing empty blocks")
	}
	if !strings.Contains(result, "(3/4)") {
		t.Fatal("missing fraction")
	}
}

func TestRenderProgress_WithChecklist(t *testing.T) {
	data := `{"done":2,"total":3,"items":["task A","task B","task C"]}`
	result := renderProgress(data, 80)

	if !strings.Contains(result, "☑") {
		t.Fatal("missing done checkmark")
	}
	if !strings.Contains(result, "☐") {
		t.Fatal("missing pending checkbox")
	}
	if !strings.Contains(result, "task A") || !strings.Contains(result, "task C") {
		t.Fatal("missing task items")
	}
}

func TestRenderProgress_ZeroTotal(t *testing.T) {
	data := `{"done":0,"total":0}`
	// Should not panic on divide-by-zero.
	result := renderProgress(data, 80)
	if result == "" {
		t.Fatal("should produce output even with zero total")
	}
}

func TestRenderChart_Sparkline(t *testing.T) {
	data := `{"kind":"sparkline","values":[1,3,5,7,2,4,6,8]}`
	result := renderChart(data, 80)

	// Should contain sparkline block characters.
	hasBlocks := false
	for _, r := range result {
		if r >= '▁' && r <= '█' {
			hasBlocks = true
			break
		}
	}
	if !hasBlocks {
		t.Fatalf("result = %q, missing sparkline blocks", result)
	}
}

func TestRenderChart_BarChart(t *testing.T) {
	data := `{"kind":"bar","values":[10,20,30],"labels":["A","B","C"]}`
	result := renderChart(data, 80)

	if !strings.Contains(result, "A") || !strings.Contains(result, "C") {
		t.Fatal("missing labels")
	}
	if !strings.Contains(result, "█") {
		t.Fatal("missing bar blocks")
	}
}

func TestRenderPanel_BorderedWrapper(t *testing.T) {
	msg := RenderPanelMsg{
		Type:  "progress",
		Title: "Sprint",
		Data:  `{"done":1,"total":2}`,
	}
	result := RenderPanel(msg, 60)

	if !strings.Contains(result, "╭") {
		t.Fatal("missing top border")
	}
	if !strings.Contains(result, "╰") {
		t.Fatal("missing bottom border")
	}
	if !strings.Contains(result, "Sprint") {
		t.Fatal("missing title")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BLUE: Edge cases — boundary conditions
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTable_EmptyRows(t *testing.T) {
	data := `{"columns":["A","B"],"rows":[]}`
	result := renderTable(data, 80)
	// Should have header but no data rows.
	if !strings.Contains(result, "A") {
		t.Fatal("missing header")
	}
}

func TestRenderTable_EmptyColumns(t *testing.T) {
	data := `{"columns":[],"rows":[]}`
	result := renderTable(data, 80)
	if !strings.Contains(result, "empty table") {
		t.Fatalf("result = %q, want empty table message", result)
	}
}

func TestRenderTable_SingleColumn(t *testing.T) {
	data := `{"columns":["Only"],"rows":[["val"]]}`
	result := renderTable(data, 80)
	// Should NOT have separator character │.
	if strings.Contains(result, "│") {
		t.Fatal("single column should not have │ separator")
	}
}

func TestRenderTree_LeafOnly(t *testing.T) {
	data := `{"root":{"label":"leaf"}}`
	result := renderTree(data, 80)
	if !strings.Contains(result, "leaf") {
		t.Fatal("missing leaf")
	}
	// No connectors for leaf-only tree.
	if strings.Contains(result, "├") || strings.Contains(result, "└") {
		t.Fatal("leaf-only should have no connectors")
	}
}

func TestRenderProgress_100Percent(t *testing.T) {
	data := `{"done":5,"total":5,"items":["a","b","c","d","e"]}`
	result := renderProgress(data, 80)
	if !strings.Contains(result, "100") {
		t.Fatal("missing 100%")
	}
	// All items should be checked.
	if strings.Contains(result, "☐") {
		t.Fatal("100% should have no pending items")
	}
}

func TestRenderChart_SingleValue(t *testing.T) {
	data := `{"kind":"sparkline","values":[42]}`
	result := renderChart(data, 80)
	if result == "" {
		t.Fatal("should render single value")
	}
}

func TestRenderChart_AllSameValues(t *testing.T) {
	data := `{"kind":"sparkline","values":[5,5,5,5]}`
	result := renderChart(data, 80)
	// All same → should render (no division by zero in normalization).
	if result == "" {
		t.Fatal("should render flat sparkline")
	}
}

func TestRenderChart_EmptyValues(t *testing.T) {
	data := `{"kind":"sparkline","values":[]}`
	result := renderChart(data, 80)
	if !strings.Contains(result, "no data") {
		t.Fatalf("result = %q, want no data message", result)
	}
}

func TestRenderPanel_NarrowWidth(t *testing.T) {
	msg := RenderPanelMsg{Type: "table", Title: "T", Data: `{"columns":["A"],"rows":[["1"]]}`}
	// Width below minimum should be clamped.
	result := RenderPanel(msg, 5)
	if result == "" {
		t.Fatal("should render even at narrow width")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Diff renderer
// ═══════════════════════════════════════════════════════════════════════

func TestRenderDiffPanel_BasicDiff(t *testing.T) {
	data := `{"file":"main.go","hunks":["@@ -1,3 +1,4 @@\n package main\n+import \"fmt\"\n func main() {}"]}`
	result := renderDiffPanel(data)
	if !strings.Contains(result, "main.go") {
		t.Fatal("missing filename")
	}
	if !strings.Contains(result, "import") {
		t.Fatal("missing diff content")
	}
}

func TestRenderDiffPanel_InvalidJSON(t *testing.T) {
	result := renderDiffPanel("not json")
	if !strings.Contains(result, "invalid diff data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderDiffPanel_EmptyHunks(t *testing.T) {
	data := `{"file":"empty.go","hunks":[]}`
	result := renderDiffPanel(data)
	if !strings.Contains(result, "empty.go") {
		t.Fatal("missing filename even with empty hunks")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Diagram renderer
// ═══════════════════════════════════════════════════════════════════════

func TestRenderDiagram_TwoNodes(t *testing.T) {
	data := `{"nodes":[{"id":"a","label":"app"},{"id":"b","label":"staff"}],"edges":[{"from":"a","to":"b"}]}`
	result := renderDiagram(data, 80)
	if !strings.Contains(result, "app") || !strings.Contains(result, "staff") {
		t.Fatal("missing node labels")
	}
	if !strings.Contains(result, "──→") {
		t.Fatal("missing arrow")
	}
	if !strings.Contains(result, "┌") {
		t.Fatal("missing box border")
	}
}

func TestRenderDiagram_InvalidJSON(t *testing.T) {
	result := renderDiagram("bad", 80)
	if !strings.Contains(result, "invalid diagram data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderDiagram_EmptyNodes(t *testing.T) {
	result := renderDiagram(`{"nodes":[],"edges":[]}`, 80)
	if !strings.Contains(result, "empty diagram") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderDiagram_EdgeLabel(t *testing.T) {
	data := `{"nodes":[{"id":"a","label":"X"},{"id":"b","label":"Y"}],"edges":[{"from":"a","to":"b","label":"uses"}]}`
	result := renderDiagram(data, 80)
	if !strings.Contains(result, "uses") {
		t.Fatal("missing edge label")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Timeline renderer
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTimeline_ThreeEvents(t *testing.T) {
	data := `{"events":[{"label":"Build","state":"done","time":"0:05"},{"label":"Test","state":"active","time":"0:10"},{"label":"Deploy","state":"pending"}]}`
	result := renderTimeline(data)
	if !strings.Contains(result, "Build") || !strings.Contains(result, "Deploy") {
		t.Fatal("missing event labels")
	}
	if !strings.Contains(result, "●") {
		t.Fatal("missing filled marker")
	}
	if !strings.Contains(result, "○") {
		t.Fatal("missing empty marker")
	}
}

func TestRenderTimeline_InvalidJSON(t *testing.T) {
	result := renderTimeline("bad")
	if !strings.Contains(result, "invalid timeline data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderTimeline_EmptyEvents(t *testing.T) {
	result := renderTimeline(`{"events":[]}`)
	if !strings.Contains(result, "no events") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderTimeline_ErrorState(t *testing.T) {
	data := `{"events":[{"label":"Crashed","state":"error"}]}`
	result := renderTimeline(data)
	if !strings.Contains(result, "●") {
		t.Fatal("error state should have filled marker")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Code renderer
// ═══════════════════════════════════════════════════════════════════════

func TestRenderCode_BasicSource(t *testing.T) {
	data := `{"language":"go","source":"package main\n\nfunc main() {}"}`
	result := renderCode(data, 80)
	if !strings.Contains(result, "package main") {
		t.Fatal("missing source content")
	}
	if !strings.Contains(result, "  1 ") {
		t.Fatal("missing line numbers")
	}
}

func TestRenderCode_InvalidJSON(t *testing.T) {
	result := renderCode("bad", 80)
	if !strings.Contains(result, "invalid code data") {
		t.Fatalf("result = %q", result)
	}
}

func TestRenderCode_EmptySource(t *testing.T) {
	result := renderCode(`{"language":"go","source":""}`, 80)
	if !strings.Contains(result, "empty source") {
		t.Fatalf("result = %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// All 8 types through RenderPanel wrapper
// ═══════════════════════════════════════════════════════════════════════

func TestRenderPanel_All8Types(t *testing.T) {
	types := map[string]string{
		"table":    `{"columns":["A"],"rows":[["1"]]}`,
		"tree":     `{"root":{"label":"root"}}`,
		"progress": `{"done":1,"total":2}`,
		"chart":    `{"kind":"sparkline","values":[1,2,3]}`,
		"diff":     `{"file":"f.go","hunks":["+ added"]}`,
		"diagram":  `{"nodes":[{"id":"a","label":"A"}],"edges":[]}`,
		"timeline": `{"events":[{"label":"E","state":"done"}]}`,
		"code":     `{"language":"go","source":"x := 1"}`,
	}
	for typ, data := range types {
		t.Run(typ, func(t *testing.T) {
			msg := RenderPanelMsg{Type: typ, Title: "Test " + typ, Data: data}
			result := RenderPanel(msg, 60)
			if result == "" {
				t.Fatalf("empty result for type %q", typ)
			}
			if !strings.Contains(result, "╭") {
				t.Fatalf("missing border for type %q", typ)
			}
		})
	}
}

func TestPad(t *testing.T) {
	if pad("hi", 5) != "hi   " {
		t.Fatalf("pad = %q", pad("hi", 5))
	}
	if pad("hello", 3) != "hello" {
		t.Fatalf("pad should not truncate: %q", pad("hello", 3))
	}
}
