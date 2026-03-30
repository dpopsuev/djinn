// plan_adapter.go — Converts ArtifactGraph to DataFlowGraph for diagram rendering (GOL-61).
//
// Like CircuitToGraph() but for artifact dependency visualization.
// Artifacts become nodes (status → Changed flag). DependsOn become edges.
package render

import (
	"github.com/dpopsuev/djinn/artifact"
)

// ArtifactGraphToDataFlow converts an artifact.Graph into a DataFlowGraph
// for rendering with the diagram renderer. Artifacts become nodes;
// DependsOn edges become directed edges.
func ArtifactGraphToDataFlow(g *artifact.Graph, title string) *DataFlowGraph {
	dfg := NewGraph(title)

	all := g.ListSorted()
	for i := range all {
		a := &all[i]
		dfg.AddNode(Node{
			ID:          a.ID,
			Name:        a.Title,
			Kind:        NodePackage, // artifacts are structural units
			Package:     a.Kind,
			Changed:     a.Status == artifact.StatusInProgress || a.Status == artifact.StatusActive,
			PassThrough: a.Status == artifact.StatusComplete || a.Status == artifact.StatusDone,
		})
	}

	for i := range all {
		a := &all[i]
		for _, depID := range a.DependsOn {
			dfg.AddEdge(Edge{
				From:  depID,
				To:    a.ID,
				Label: string(a.Status),
			})
		}
	}

	return dfg
}
