// mermaid.go — Mermaid flowchart renderer (TSK-432).
package render

import (
	"fmt"
	"strings"
)

type mermaidRenderer struct{}

// NewMermaidRenderer creates a renderer that outputs Mermaid flowchart syntax.
func NewMermaidRenderer() Renderer {
	return &mermaidRenderer{}
}

func (r *mermaidRenderer) Render(g *DataFlowGraph, _ RenderOpts) (string, error) {
	if g == nil || len(g.Nodes) == 0 {
		return "", nil
	}

	var b strings.Builder
	b.WriteString("graph TD\n")

	// Class definitions for styling.
	b.WriteString("    classDef changed fill:#f9a8a8,stroke:#ee0000,color:#3f0000\n")
	b.WriteString("    classDef passthrough fill:#f2f2f2,stroke:#c7c7c7,color:#707070\n")

	// Nodes.
	for i := range g.Nodes {
		n := &g.Nodes[i]
		label := n.Name
		if n.Signature != "" {
			label += "<br/>" + n.Signature
		}
		fmt.Fprintf(&b, "    %s[\"%s\"]", n.ID, label)
		if n.Changed {
			b.WriteString(":::changed")
		} else if n.PassThrough {
			b.WriteString(":::passthrough")
		}
		b.WriteByte('\n')
	}

	// Edges.
	for i := range g.Edges {
		e := &g.Edges[i]
		arrow := " --> "
		if e.Violation {
			arrow = " -.-> "
		}
		if e.Label != "" {
			fmt.Fprintf(&b, "    %s%s|\"%s\"| %s\n", e.From, arrow, e.Label, e.To)
		} else {
			fmt.Fprintf(&b, "    %s%s%s\n", e.From, arrow, e.To)
		}
	}

	return b.String(), nil
}
