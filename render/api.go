// api.go — Public API for building DataFlowGraphs from external data (TSK-434).
//
// Provides builder functions and the CircuitStop contract used by the
// Visual Reviewer (GOL-50, Wave 3) to convert circuits into renderable graphs.
package render

// CircuitStop represents a single stop on a data flow circuit.
// Defined here so render/ owns the contract; review/ produces them.
type CircuitStop struct {
	ID              string
	Name            string
	Package         string
	SignatureBefore string
	SignatureAfter  string
	Changed         bool
	PassThrough     bool
	InputType       string
	OutputType      string
}

// CircuitToGraph converts a sequence of circuit stops into a DataFlowGraph.
// Each stop becomes a node; consecutive stops are connected by edges.
func CircuitToGraph(title string, stops []CircuitStop) *DataFlowGraph {
	g := NewGraph(title)

	for i := range stops {
		s := &stops[i]
		sig := s.SignatureAfter
		if sig == "" {
			sig = s.SignatureBefore
		}
		g.AddNode(Node{
			ID:          s.ID,
			Name:        s.Name,
			Kind:        NodeFunction,
			Package:     s.Package,
			Signature:   sig,
			Changed:     s.Changed,
			PassThrough: s.PassThrough,
		})

		// Edge from previous stop to this one.
		if i > 0 {
			prev := stops[i-1]
			label := ""
			if s.InputType != "" {
				label = s.InputType
			}
			g.AddEdge(Edge{
				From:  prev.ID,
				To:    s.ID,
				Label: label,
			})
		}
	}

	return g
}
