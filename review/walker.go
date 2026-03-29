// walker.go — DFS circuit walker (TSK-443).
//
// Walks from an entry point through the dependency graph via DFS,
// producing ordered CircuitStops. Changed nodes are flagged, unmodified
// nodes on the path are marked as pass-through.
package review

import (
	"github.com/dpopsuev/djinn/render"
)

// WalkCircuit performs DFS from an entry point, producing circuit stops.
func WalkCircuit(entry EntryPoint, deps []Dependency, changedSet map[string]bool) []render.CircuitStop {
	// Build adjacency list.
	adj := make(map[string][]Dependency)
	for i := range deps {
		adj[deps[i].From] = append(adj[deps[i].From], deps[i])
	}

	var stops []render.CircuitStop
	visited := make(map[string]bool)

	var dfs func(node string)
	dfs = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true

		pkg, name := splitNode(node)
		stops = append(stops, render.CircuitStop{
			ID:          node,
			Name:        name,
			Package:     pkg,
			Changed:     changedSet[node],
			PassThrough: !changedSet[node],
		})

		for _, dep := range adj[node] {
			dfs(dep.To)
		}
	}

	dfs(entry.Name)
	return stops
}

// splitNode separates "package.Function" into package and function name.
func splitNode(node string) (pkg, name string) {
	for i := len(node) - 1; i >= 0; i-- {
		if node[i] == '.' {
			return node[:i], node[i+1:]
		}
	}
	return "", node
}
