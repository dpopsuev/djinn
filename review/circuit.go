// circuit.go — Circuit extraction + entry point detection (TSK-443).
//
// A circuit is a complete data flow path from an entry point through the
// dependency graph. Each circuit is a self-contained unit of review —
// no pre-flight understanding needed.
package review

import (
	"strings"

	"github.com/dpopsuev/djinn/render"
)

// Circuit represents a complete data flow path from entry point to exit.
type Circuit struct {
	Title      string
	EntryPoint string
	Stops      []render.CircuitStop
	Shared     []string // stop IDs also on other circuits
}

// EntryPoint represents a detected entry point in the codebase.
type EntryPoint struct {
	Name    string
	Package string
	Kind    string // "handler", "cmd", "worker", "main", "server"
}

// Dependency represents a directed edge in the dependency graph.
type Dependency struct {
	From    string // "package.Function"
	To      string
	Package string
}

// entryKeywords used to identify entry point packages/functions.
var entryKeywords = []struct {
	keyword string
	kind    string
}{
	{"handler", "handler"},
	{"server", "server"},
	{"cmd", "cmd"},
	{"main", "main"},
	{"worker", "worker"},
	{"listener", "listener"},
	{"grpc", "handler"},
	{"http", "handler"},
	{"api", "handler"},
}

// DetectEntryPoints finds entry points from dependency graph edges.
func DetectEntryPoints(deps []Dependency) []EntryPoint {
	seen := make(map[string]bool)
	var entries []EntryPoint

	// Collect all unique "from" nodes.
	fromNodes := make(map[string]string) // node → package
	for i := range deps {
		fromNodes[deps[i].From] = deps[i].Package
	}

	for node, pkg := range fromNodes {
		if seen[node] {
			continue
		}
		for _, kw := range entryKeywords {
			if matchesEntryKeyword(node, kw.keyword) || matchesEntryKeyword(pkg, kw.keyword) {
				seen[node] = true
				entries = append(entries, EntryPoint{
					Name:    node,
					Package: pkg,
					Kind:    kw.kind,
				})
				break
			}
		}
	}
	return entries
}

// ExtractCircuits identifies affected entry points and walks their call graphs.
// Only entry points with changed code on their path are included.
func ExtractCircuits(changedFiles []string, deps []Dependency) []Circuit {
	entries := DetectEntryPoints(deps)
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	circuits := make([]Circuit, 0, len(entries))
	for _, entry := range entries {
		stops := WalkCircuit(entry, deps, changedSet)
		if !hasChangedStop(stops) {
			continue // skip circuits with no changed code
		}
		circuits = append(circuits, Circuit{
			Title:      entry.Kind + ": " + entry.Name,
			EntryPoint: entry.Name,
			Stops:      stops,
		})
	}

	// Mark shared stops across circuits.
	markSharedStops(circuits)
	return circuits
}

func hasChangedStop(stops []render.CircuitStop) bool {
	for i := range stops {
		if stops[i].Changed {
			return true
		}
	}
	return false
}

func markSharedStops(circuits []Circuit) {
	stopCircuits := make(map[string][]int) // stop ID → circuit indices
	for ci, c := range circuits {
		for _, s := range c.Stops {
			stopCircuits[s.ID] = append(stopCircuits[s.ID], ci)
		}
	}
	for ci := range circuits {
		for _, s := range circuits[ci].Stops {
			indices := stopCircuits[s.ID]
			if len(indices) > 1 {
				circuits[ci].Shared = append(circuits[ci].Shared, s.ID)
			}
		}
	}
}

// matchesEntryKeyword checks if a node or package name matches an entry keyword
// using word boundary logic (not substring). Matches: "handler.Create", "cmd/main",
// "worker_pool". Does NOT match: "domain" containing "main".
func matchesEntryKeyword(s, keyword string) bool {
	s = strings.ToLower(s)
	idx := strings.Index(s, keyword)
	if idx < 0 {
		return false
	}
	// Check left boundary: start of string or non-alphanumeric.
	if idx > 0 {
		prev := s[idx-1]
		if isAlphaNum(prev) {
			return false
		}
	}
	// Check right boundary: end of string or non-alphanumeric.
	end := idx + len(keyword)
	if end < len(s) {
		next := s[end]
		if isAlphaNum(next) {
			return false
		}
	}
	return true
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
