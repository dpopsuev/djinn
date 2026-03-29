// graph.go — DataFlowGraph domain model for circuit-based code review (TSK-430).
//
// Pure value types with no external dependencies. Represents a data flow graph
// as nodes (functions, types, packages) and directed edges (calls, data flow).
// Used by the Visual Reviewer to render circuit-based code review tours.
package render

// NodeKind classifies what a graph node represents.
type NodeKind string

const (
	NodeFunction  NodeKind = "function"
	NodeType      NodeKind = "type"
	NodePackage   NodeKind = "package"
	NodeInterface NodeKind = "interface"
)

// Node is a single element in a data flow graph.
type Node struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Kind        NodeKind          `json:"kind"`
	Package     string            `json:"package,omitempty"`
	Signature   string            `json:"signature,omitempty"` // function sig or type def
	Changed     bool              `json:"changed,omitempty"`   // highlighted in review
	PassThrough bool              `json:"pass_through,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	From      string `json:"from"`                // Node.ID
	To        string `json:"to"`                  // Node.ID
	Label     string `json:"label,omitempty"`     // data type or relationship
	Violation bool   `json:"violation,omitempty"` // architectural violation
}

// DataFlowGraph represents a complete data flow circuit.
type DataFlowGraph struct {
	Title string `json:"title"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// NewGraph creates an empty graph with the given title.
func NewGraph(title string) *DataFlowGraph {
	return &DataFlowGraph{
		Title: title,
		Nodes: make([]Node, 0),
		Edges: make([]Edge, 0),
	}
}

// AddNode appends a node to the graph.
func (g *DataFlowGraph) AddNode(n Node) {
	g.Nodes = append(g.Nodes, n)
}

// AddEdge appends an edge to the graph.
func (g *DataFlowGraph) AddEdge(e Edge) {
	g.Edges = append(g.Edges, e)
}

// NodeByID returns the node with the given ID and true, or zero value and false.
func (g *DataFlowGraph) NodeByID(id string) (Node, bool) {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return g.Nodes[i], true
		}
	}
	return Node{}, false
}

// ChangedNodes returns only nodes marked as changed.
func (g *DataFlowGraph) ChangedNodes() []Node {
	out := make([]Node, 0, len(g.Nodes))
	for i := range g.Nodes {
		if g.Nodes[i].Changed {
			out = append(out, g.Nodes[i])
		}
	}
	return out
}
