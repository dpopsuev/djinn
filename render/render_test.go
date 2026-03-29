package render

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleGraph() *DataFlowGraph {
	g := NewGraph("POST /orders")
	g.AddNode(Node{ID: "h", Name: "CreateOrder", Kind: NodeFunction, Package: "handler", Changed: true, Signature: "(req) (resp, error)"})
	g.AddNode(Node{ID: "v", Name: "Validate", Kind: NodeFunction, Package: "validator", Changed: true})
	g.AddNode(Node{ID: "d", Name: "Order.New", Kind: NodeFunction, Package: "domain", Changed: false, PassThrough: true})
	g.AddNode(Node{ID: "r", Name: "Save", Kind: NodeFunction, Package: "repo", Changed: true})
	g.AddNode(Node{ID: "p", Name: "Insert", Kind: NodeFunction, Package: "postgres", Changed: false})
	g.AddEdge(Edge{From: "h", To: "v", Label: "OrderRequest"})
	g.AddEdge(Edge{From: "v", To: "d"})
	g.AddEdge(Edge{From: "d", To: "r"})
	g.AddEdge(Edge{From: "r", To: "p", Label: "Order", Violation: true})
	return g
}

func TestNewGraph(t *testing.T) {
	g := NewGraph("test")
	if g.Title != "test" {
		t.Errorf("title = %q, want %q", g.Title, "test")
	}
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Error("new graph should be empty")
	}
}

func TestNodeByID(t *testing.T) {
	g := sampleGraph()
	n, ok := g.NodeByID("v")
	if !ok {
		t.Fatal("node 'v' not found")
	}
	if n.Name != "Validate" {
		t.Errorf("name = %q, want %q", n.Name, "Validate")
	}
	_, ok = g.NodeByID("nonexistent")
	if ok {
		t.Error("nonexistent node should not be found")
	}
}

func TestChangedNodes(t *testing.T) {
	g := sampleGraph()
	changed := g.ChangedNodes()
	if len(changed) != 3 {
		t.Errorf("changed = %d, want 3", len(changed))
	}
}

func TestTUIRenderer(t *testing.T) {
	g := sampleGraph()
	r := NewTUIRenderer()
	out, err := r.Render(g, RenderOpts{Highlight: true})
	if err != nil {
		t.Fatal(err)
	}

	// Verify structure.
	if !strings.Contains(out, "POST /orders") {
		t.Error("missing title")
	}
	if !strings.Contains(out, "● handler.CreateOrder") {
		t.Error("changed node should have ● prefix")
	}
	if !strings.Contains(out, "○ domain.Order.New") {
		t.Error("pass-through node should have ○ prefix")
	}
	if !strings.Contains(out, "⚠→") {
		t.Error("violation edge should have ⚠→")
	}
	if !strings.Contains(out, "(OrderRequest)") {
		t.Error("edge label should show data type")
	}
}

func TestTUIRendererEmpty(t *testing.T) {
	r := NewTUIRenderer()
	out, err := r.Render(NewGraph("empty"), RenderOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("empty graph should produce empty output, got %q", out)
	}
}

func TestTUIRendererCollapse(t *testing.T) {
	g := NewGraph("big")
	for i := range 20 {
		g.AddNode(Node{ID: strings.Repeat("n", i+1), Name: "Func"})
	}
	r := NewTUIRenderer()
	out, err := r.Render(g, RenderOpts{CollapseAt: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "15 more") {
		t.Errorf("should show collapsed count, got:\n%s", out)
	}
}

func TestMermaidRenderer(t *testing.T) {
	g := sampleGraph()
	r := NewMermaidRenderer()
	out, err := r.Render(g, RenderOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(out, "graph TD") {
		t.Error("Mermaid should start with 'graph TD'")
	}
	if !strings.Contains(out, ":::changed") {
		t.Error("changed nodes should have :::changed class")
	}
	if !strings.Contains(out, ":::passthrough") {
		t.Error("pass-through nodes should have :::passthrough class")
	}
	if !strings.Contains(out, "-.->") {
		t.Error("violation edges should use dashed arrows")
	}
}

func TestJSONRendererRoundTrip(t *testing.T) {
	g := sampleGraph()
	r := NewJSONRenderer()
	out, err := r.Render(g, RenderOpts{})
	if err != nil {
		t.Fatal(err)
	}

	var parsed DataFlowGraph
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("JSON output not parseable: %v", err)
	}
	if parsed.Title != g.Title {
		t.Errorf("title = %q, want %q", parsed.Title, g.Title)
	}
	if len(parsed.Nodes) != len(g.Nodes) {
		t.Errorf("nodes = %d, want %d", len(parsed.Nodes), len(g.Nodes))
	}
	if len(parsed.Edges) != len(g.Edges) {
		t.Errorf("edges = %d, want %d", len(parsed.Edges), len(g.Edges))
	}
}

func TestCircuitToGraph(t *testing.T) {
	stops := []CircuitStop{
		{ID: "1", Name: "CreateOrder", Package: "handler", Changed: true, InputType: "OrderRequest"},
		{ID: "2", Name: "Validate", Package: "validator", Changed: true, InputType: "OrderRequest"},
		{ID: "3", Name: "Save", Package: "repo", Changed: false, PassThrough: true},
	}
	g := CircuitToGraph("POST /orders", stops)

	if len(g.Nodes) != 3 {
		t.Errorf("nodes = %d, want 3", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Errorf("edges = %d, want 2", len(g.Edges))
	}
	if g.Nodes[0].Changed != true {
		t.Error("first node should be changed")
	}
	if g.Nodes[2].PassThrough != true {
		t.Error("third node should be pass-through")
	}
	if g.Edges[0].Label != "OrderRequest" {
		t.Errorf("first edge label = %q, want %q", g.Edges[0].Label, "OrderRequest")
	}
}
