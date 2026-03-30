package render

import (
	"testing"

	"github.com/dpopsuev/djinn/artifact"
)

func testArtifactGraph() *artifact.Graph {
	reg := artifact.DefaultRegistry()
	g := artifact.NewGraph("test-plan", reg)

	g.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "Hub Mediators", Status: artifact.StatusComplete})                                  //nolint:errcheck,gocritic // test setup
	g.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "Planner Port", Status: artifact.StatusComplete, DependsOn: []string{"seg-1"}})     //nolint:errcheck,gocritic // test setup
	g.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "Shell Maturity", Status: artifact.StatusInProgress, DependsOn: []string{"seg-2"}}) //nolint:errcheck,gocritic // test setup

	return g
}

func TestArtifactGraphToDataFlow_Nodes(t *testing.T) {
	g := testArtifactGraph()
	dfg := ArtifactGraphToDataFlow(g, "Plan Dependencies")

	if dfg.Title != "Plan Dependencies" {
		t.Errorf("title = %q, want 'Plan Dependencies'", dfg.Title)
	}

	if len(dfg.Nodes) != 3 { //nolint:mnd // 3 artifacts
		t.Fatalf("nodes = %d, want 3", len(dfg.Nodes))
	}

	// Find nodes by ID.
	nodeMap := make(map[string]*Node, len(dfg.Nodes))
	for i := range dfg.Nodes {
		nodeMap[dfg.Nodes[i].ID] = &dfg.Nodes[i]
	}

	hub := nodeMap["seg-1"]
	if hub == nil || hub.Name != "Hub Mediators" {
		t.Error("node seg-1 should be 'Hub Mediators'")
	}
	if !hub.PassThrough {
		t.Error("complete artifact should be PassThrough")
	}

	shell := nodeMap["seg-3"]
	if shell == nil || shell.Name != "Shell Maturity" {
		t.Error("node seg-3 should be 'Shell Maturity'")
	}
	if !shell.Changed {
		t.Error("in_progress artifact should be Changed")
	}
}

func TestArtifactGraphToDataFlow_Edges(t *testing.T) {
	g := testArtifactGraph()
	dfg := ArtifactGraphToDataFlow(g, "deps")

	if len(dfg.Edges) != 2 { //nolint:mnd // 2 dependency edges
		t.Fatalf("edges = %d, want 2", len(dfg.Edges))
	}

	// seg-1 → seg-2 (Hub → Planner)
	found := false
	for _, e := range dfg.Edges {
		if e.From == "seg-1" && e.To == "seg-2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected edge seg-1 → seg-2")
	}
}

func TestArtifactGraphToDataFlow_EmptyGraph(t *testing.T) {
	g := artifact.NewGraph("empty", artifact.DefaultRegistry())
	dfg := ArtifactGraphToDataFlow(g, "empty")

	if len(dfg.Nodes) != 0 || len(dfg.Edges) != 0 {
		t.Error("empty graph should produce empty DataFlowGraph")
	}
}
