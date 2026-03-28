// arch.go — import graph analyser.
//
// AnalyzeImports runs `go list -json ./...` and builds a package-level
// dependency graph. It detects import cycles via Tarjan's SCC algorithm
// and computes fan-in/fan-out for each internal package.
//
// CheckLayerViolations verifies that packages in lower architectural
// layers do not import packages from higher layers.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// PackageInfo describes a single Go package.
type PackageInfo struct {
	Path    string   `json:"path"`
	Name    string   `json:"name"`
	Imports []string `json:"imports"`
	Files   int      `json:"files"`
}

// ArchReport is the result of import graph analysis.
type ArchReport struct {
	Packages []PackageInfo  `json:"packages"`
	Cycles   [][]string     `json:"cycles,omitempty"`
	FanIn    map[string]int `json:"fan_in"`
	FanOut   map[string]int `json:"fan_out"`
}

// goListPackage is the subset of `go list -json` output we parse.
type goListPackage struct {
	ImportPath  string   `json:"ImportPath"`
	Name        string   `json:"Name"`
	Imports     []string `json:"Imports"`
	GoFiles     []string `json:"GoFiles"`
	TestGoFiles []string `json:"TestGoFiles,omitempty"`
}

// AnalyzeImports runs `go list -json ./...` in dir, builds the import
// graph, detects cycles, and computes fan-in/fan-out.
func AnalyzeImports(ctx context.Context, dir string) (*ArchReport, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-json", "./...")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	pkgs, err := parseGoListJSON(out)
	if err != nil {
		return nil, err
	}

	// Build set of internal packages (those under the module).
	internal := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		internal[p.ImportPath] = true
	}

	report := &ArchReport{
		FanIn:  make(map[string]int),
		FanOut: make(map[string]int),
	}

	// Build adjacency list (only internal edges).
	adj := make(map[string][]string, len(pkgs))
	for _, p := range pkgs {
		fileCount := len(p.GoFiles)
		var internalImports []string
		for _, imp := range p.Imports {
			if internal[imp] {
				internalImports = append(internalImports, imp)
				report.FanIn[imp]++
			}
		}

		report.FanOut[p.ImportPath] = len(internalImports)
		adj[p.ImportPath] = internalImports

		report.Packages = append(report.Packages, PackageInfo{
			Path:    p.ImportPath,
			Name:    p.Name,
			Imports: internalImports,
			Files:   fileCount,
		})
	}

	// Ensure every internal package has a FanIn entry.
	for pkg := range internal {
		if _, ok := report.FanIn[pkg]; !ok {
			report.FanIn[pkg] = 0
		}
	}

	// Detect cycles via Tarjan's SCC.
	report.Cycles = findCycles(adj)

	sort.Slice(report.Packages, func(i, j int) bool {
		return report.Packages[i].Path < report.Packages[j].Path
	})

	return report, nil
}

// parseGoListJSON parses the concatenated JSON objects from `go list -json`.
// The output is NOT a JSON array — it's one JSON object per package.
func parseGoListJSON(data []byte) ([]goListPackage, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	var pkgs []goListPackage
	for dec.More() {
		var p goListPackage
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// findCycles uses Tarjan's SCC algorithm and returns cycles
// (SCCs with more than one node, or self-loops).
func findCycles(adj map[string][]string) [][]string {
	var (
		index   int
		stack   []string
		onStack = make(map[string]bool)
		indices = make(map[string]int)
		lowlink = make(map[string]int)
		visited = make(map[string]bool)
		sccs    [][]string
	)

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		visited[v] = true
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if !visited[w] {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		// If v is a root node, pop SCC.
		if lowlink[v] == indices[v] { //nolint:nestif // Tarjan's algorithm requires this nesting
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			// Only report SCCs that indicate cycles (size > 1,
			// or size == 1 with self-loop).
			if len(scc) > 1 {
				sort.Strings(scc)
				sccs = append(sccs, scc)
			} else if len(scc) == 1 {
				for _, dep := range adj[scc[0]] {
					if dep == scc[0] {
						sccs = append(sccs, scc)
						break
					}
				}
			}
		}
	}

	// Process all nodes.
	nodes := make([]string, 0, len(adj))
	for n := range adj {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes) // deterministic
	for _, n := range nodes {
		if !visited[n] {
			strongConnect(n)
		}
	}

	return sccs
}

// CheckLayerViolations checks that packages in lower layers do not
// import packages from higher layers.
//
// layers is an ordered list from lowest to highest, e.g.:
//
//	["domain", "port", "adapter"]
//
// A package containing "domain" in its path importing a package containing
// "adapter" in its path is a violation.
//
// Returns a list of violation descriptions.
func CheckLayerViolations(report *ArchReport, layers []string) []string {
	if len(layers) < 2 {
		return nil
	}

	// Build a layer-index map: lower index = lower layer.
	layerIdx := make(map[string]int, len(layers))
	for i, l := range layers {
		layerIdx[l] = i
	}

	// Determine the layer of a package path by checking which layer
	// keyword appears in the path. If multiple match, use the first.
	packageLayer := func(path string) (string, int, bool) {
		for _, l := range layers {
			if strings.Contains(path, l) {
				return l, layerIdx[l], true
			}
		}
		return "", -1, false
	}

	var violations []string
	for _, pkg := range report.Packages {
		srcLayer, srcIdx, srcOK := packageLayer(pkg.Path)
		if !srcOK {
			continue
		}
		for _, imp := range pkg.Imports {
			dstLayer, dstIdx, dstOK := packageLayer(imp)
			if !dstOK {
				continue
			}
			if srcIdx < dstIdx {
				violations = append(violations, fmt.Sprintf(
					"%s (%s layer) imports %s (%s layer)",
					pkg.Path, srcLayer, imp, dstLayer,
				))
			}
		}
	}

	sort.Strings(violations)
	return violations
}
