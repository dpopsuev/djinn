// renderer.go — Renderer interface + TUI box-drawing renderer (TSK-431).
//
// Strategy pattern: same DataFlowGraph renders to TUI, Mermaid, or JSON
// via different Renderer implementations.
package render

import (
	"fmt"
	"strings"
)

// Format identifies a rendering output format.
type Format string

const (
	FormatTUI     Format = "tui"
	FormatMermaid Format = "mermaid"
	FormatJSON    Format = "json"
)

// RenderOpts controls rendering behavior.
type RenderOpts struct {
	Width      int  // terminal width (0 = no wrapping)
	Highlight  bool // emphasize changed nodes
	CollapseAt int  // collapse after N nodes (0 = no collapse)
}

// Renderer produces string output from a DataFlowGraph.
type Renderer interface {
	Render(g *DataFlowGraph, opts RenderOpts) (string, error)
}

// tuiRenderer renders graphs as box-drawing text for terminal display.
type tuiRenderer struct{}

// NewTUIRenderer creates a TUI renderer using box-drawing characters.
func NewTUIRenderer() Renderer {
	return &tuiRenderer{}
}

func (r *tuiRenderer) Render(g *DataFlowGraph, opts RenderOpts) (string, error) {
	if g == nil || len(g.Nodes) == 0 {
		return "", nil
	}

	var b strings.Builder

	if g.Title != "" {
		fmt.Fprintf(&b, "╭─ %s ─╮\n", g.Title)
	}

	// Build adjacency for edge rendering.
	outEdges := make(map[string][]Edge)
	for i := range g.Edges {
		outEdges[g.Edges[i].From] = append(outEdges[g.Edges[i].From], g.Edges[i])
	}

	limit := len(g.Nodes)
	collapsed := 0
	if opts.CollapseAt > 0 && len(g.Nodes) > opts.CollapseAt {
		limit = opts.CollapseAt
		collapsed = len(g.Nodes) - opts.CollapseAt
	}

	for i := 0; i < limit; i++ {
		node := g.Nodes[i]
		label := r.formatNode(node, opts)
		b.WriteString(label)
		b.WriteByte('\n')

		// Render outgoing edges from this node.
		for _, edge := range outEdges[node.ID] {
			arrow := "  → "
			if edge.Violation {
				arrow = "  ⚠→ "
			}
			edgeLabel := edge.To
			if edge.Label != "" {
				edgeLabel += " (" + edge.Label + ")"
			}
			b.WriteString(arrow + edgeLabel + "\n")
		}
	}

	if collapsed > 0 {
		fmt.Fprintf(&b, "  ... and %d more nodes\n", collapsed)
	}

	return b.String(), nil
}

func (r *tuiRenderer) formatNode(n Node, opts RenderOpts) string {
	prefix := "  "
	if n.Changed && opts.Highlight {
		prefix = "● "
	} else if n.PassThrough {
		prefix = "○ "
	}

	label := n.Name
	if n.Package != "" {
		label = n.Package + "." + n.Name
	}
	if n.Signature != "" {
		label += " — " + n.Signature
	}

	return prefix + label
}
