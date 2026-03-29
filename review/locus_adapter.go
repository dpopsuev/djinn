// locus_adapter.go — Bridges Locus edge data to circuit dependencies (TSK-445).
package review

// LocusEdge represents a dependency edge from Locus scan output.
type LocusEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Weight int    `json:"weight"`
}

// DepsFromLocusEdges converts Locus edges to circuit Dependencies.
func DepsFromLocusEdges(edges []LocusEdge) []Dependency {
	deps := make([]Dependency, 0, len(edges))
	for i := range edges {
		deps = append(deps, Dependency{
			From:    edges[i].From,
			To:      edges[i].To,
			Package: edges[i].From, // Locus edges are package-level
		})
	}
	return deps
}
