package tools

import (
	"testing"
)

func TestParseGoListJSON_SinglePackage(t *testing.T) {
	input := `{"ImportPath":"example.com/foo","Name":"foo","GoFiles":["main.go"],"Imports":["fmt"]}`
	pkgs, err := parseGoListJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseGoListJSON: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("len = %d, want 1", len(pkgs))
	}
	if pkgs[0].ImportPath != "example.com/foo" {
		t.Fatalf("ImportPath = %q", pkgs[0].ImportPath)
	}
	if pkgs[0].Name != "foo" {
		t.Fatalf("Name = %q", pkgs[0].Name)
	}
	if len(pkgs[0].GoFiles) != 1 {
		t.Fatalf("GoFiles = %d, want 1", len(pkgs[0].GoFiles))
	}
}

func TestParseGoListJSON_MultiplePackages(t *testing.T) {
	input := `{"ImportPath":"example.com/a","Name":"a","GoFiles":["a.go"],"Imports":["fmt"]}
{"ImportPath":"example.com/b","Name":"b","GoFiles":["b.go","b2.go"],"Imports":["example.com/a"]}`
	pkgs, err := parseGoListJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseGoListJSON: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("len = %d, want 2", len(pkgs))
	}
}

func TestFindCycles_NoCycle(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {},
	}
	cycles := findCycles(adj)
	if len(cycles) != 0 {
		t.Fatalf("expected no cycles, got %v", cycles)
	}
}

func TestFindCycles_SimpleCycle(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}
	cycles := findCycles(adj)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %v", len(cycles), cycles)
	}
	if len(cycles[0]) != 2 {
		t.Fatalf("cycle size = %d, want 2", len(cycles[0]))
	}
}

func TestFindCycles_SelfLoop(t *testing.T) {
	adj := map[string][]string{
		"a": {"a"},
	}
	cycles := findCycles(adj)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 self-loop cycle, got %d", len(cycles))
	}
}

func TestFindCycles_ThreeNodeCycle(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}
	cycles := findCycles(adj)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}
	if len(cycles[0]) != 3 {
		t.Fatalf("cycle size = %d, want 3", len(cycles[0]))
	}
}

func TestCheckLayerViolations_NoViolation(t *testing.T) {
	report := &ArchReport{
		Packages: []PackageInfo{
			{Path: "myapp/adapter/http", Imports: []string{"myapp/domain/user"}},
			{Path: "myapp/domain/user", Imports: nil},
		},
	}
	layers := []string{"domain", "port", "adapter"}
	violations := CheckLayerViolations(report, layers)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestCheckLayerViolations_Violation(t *testing.T) {
	report := &ArchReport{
		Packages: []PackageInfo{
			{Path: "myapp/domain/user", Imports: []string{"myapp/adapter/db"}},
			{Path: "myapp/adapter/db", Imports: nil},
		},
	}
	layers := []string{"domain", "port", "adapter"}
	violations := CheckLayerViolations(report, layers)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	if violations[0] != "myapp/domain/user (domain layer) imports myapp/adapter/db (adapter layer)" {
		t.Fatalf("violation = %q", violations[0])
	}
}

func TestCheckLayerViolations_EmptyLayers(t *testing.T) {
	report := &ArchReport{
		Packages: []PackageInfo{
			{Path: "a", Imports: []string{"b"}},
		},
	}
	violations := CheckLayerViolations(report, nil)
	if len(violations) != 0 {
		t.Fatalf("expected no violations with empty layers, got %v", violations)
	}
}

func TestCheckLayerViolations_PackageNotInAnyLayer(t *testing.T) {
	report := &ArchReport{
		Packages: []PackageInfo{
			{Path: "myapp/util/helper", Imports: []string{"myapp/adapter/db"}},
		},
	}
	layers := []string{"domain", "port", "adapter"}
	violations := CheckLayerViolations(report, layers)
	// "util" is not in any layer, so no violation.
	if len(violations) != 0 {
		t.Fatalf("expected no violations for package outside layers, got %v", violations)
	}
}

func TestFanInFanOut(t *testing.T) {
	// Test fan-in/fan-out via a synthetic report built from parseGoListJSON + findCycles.
	input := `{"ImportPath":"mod/a","Name":"a","GoFiles":["a.go"],"Imports":["mod/b","mod/c"]}
{"ImportPath":"mod/b","Name":"b","GoFiles":["b.go"],"Imports":["mod/c"]}
{"ImportPath":"mod/c","Name":"c","GoFiles":["c.go"],"Imports":[]}`

	pkgs, err := parseGoListJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseGoListJSON: %v", err)
	}

	// Build report manually (since we can't call AnalyzeImports without a real module).
	internal := make(map[string]bool)
	for _, p := range pkgs {
		internal[p.ImportPath] = true
	}

	report := &ArchReport{
		FanIn:  make(map[string]int),
		FanOut: make(map[string]int),
	}

	adj := make(map[string][]string)
	for _, p := range pkgs {
		var intImports []string
		for _, imp := range p.Imports {
			if internal[imp] {
				intImports = append(intImports, imp)
				report.FanIn[imp]++
			}
		}
		report.FanOut[p.ImportPath] = len(intImports)
		adj[p.ImportPath] = intImports
		report.Packages = append(report.Packages, PackageInfo{
			Path:    p.ImportPath,
			Name:    p.Name,
			Imports: intImports,
			Files:   len(p.GoFiles),
		})
	}

	// mod/c is imported by both a and b → fan-in = 2
	if report.FanIn["mod/c"] != 2 {
		t.Fatalf("FanIn[mod/c] = %d, want 2", report.FanIn["mod/c"])
	}
	// mod/b is imported by a only → fan-in = 1
	if report.FanIn["mod/b"] != 1 {
		t.Fatalf("FanIn[mod/b] = %d, want 1", report.FanIn["mod/b"])
	}
	// mod/a imports b and c → fan-out = 2
	if report.FanOut["mod/a"] != 2 {
		t.Fatalf("FanOut[mod/a] = %d, want 2", report.FanOut["mod/a"])
	}
	// mod/c imports nothing internal → fan-out = 0
	if report.FanOut["mod/c"] != 0 {
		t.Fatalf("FanOut[mod/c] = %d, want 0", report.FanOut["mod/c"])
	}
}
