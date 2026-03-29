// tier4_complexity.go — Tier 4 heuristic: cyclomatic complexity delta (TSK-462).
//
// Counts control flow statements (if/for/switch/case/select/&&/||) in changed
// files before and after. Signals when complexity increases beyond threshold.
// Uses simple text scanning — no AST required.
package review

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ComplexityHeuristic measures cyclomatic complexity delta in changed files.
type ComplexityHeuristic struct {
	MaxComplexityDelta int
}

// NewComplexityHeuristic creates a heuristic from config.
func NewComplexityHeuristic(maxDelta int) *ComplexityHeuristic {
	return &ComplexityHeuristic{MaxComplexityDelta: maxDelta}
}

func (h *ComplexityHeuristic) Name() string { return "tier4_complexity" }

func (h *ComplexityHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if h.MaxComplexityDelta <= 0 || diff.WorkDir == "" {
		return nil, nil
	}

	totalDelta := 0
	for _, file := range diff.ChangedFiles {
		if !isSourceFile(file) {
			continue
		}
		before := complexityFromGit(diff.WorkDir, file)
		after := complexityFromFile(filepath.Join(diff.WorkDir, file))
		delta := after - before
		if delta > 0 {
			totalDelta += delta
		}
	}

	// New files contribute their full complexity.
	for _, file := range diff.AddedFiles {
		if !isSourceFile(file) {
			continue
		}
		totalDelta += complexityFromFile(filepath.Join(diff.WorkDir, file))
	}

	return []Signal{{
		Metric:    "complexity_delta",
		Value:     float64(totalDelta),
		Threshold: float64(h.MaxComplexityDelta),
		Exceeded:  totalDelta >= h.MaxComplexityDelta,
		Detail:    fmt.Sprintf("+%d complexity points", totalDelta),
	}}, nil
}

// countComplexity counts control flow keywords in source text.
func countComplexity(src string) int {
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and blank lines.
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "*") {
			continue
		}
		count += countComplexityTokens(line)
	}
	return count
}

// countComplexityTokens counts control flow tokens in a single line.
func countComplexityTokens(line string) int {
	count := 0
	// Each keyword adds a branch to the control flow.
	keywords := []string{"if ", "for ", "switch ", "case ", "select {", "catch ", "except ", "elif ", "else if "}
	for _, kw := range keywords {
		count += strings.Count(line, kw)
	}
	// Logical operators add implicit branches.
	count += strings.Count(line, " && ")
	count += strings.Count(line, " || ")
	return count
}

func complexityFromFile(path string) int {
	data, err := os.ReadFile(path) //nolint:gosec // path is from trusted DiffSnapshot
	if err != nil {
		return 0
	}
	return countComplexity(string(data))
}

func complexityFromGit(workDir, file string) int {
	cmd := exec.Command("git", "show", "HEAD:"+file) //nolint:gosec // file is from trusted DiffSnapshot
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return 0 // file didn't exist before
	}
	return countComplexity(string(out))
}

func isSourceFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".py", ".ts", ".tsx", ".js", ".jsx", ".rs", ".java",
		".c", ".cpp", ".cc", ".h", ".hpp", ".cs", ".kt", ".swift",
		".rb", ".lua", ".zig":
		return true
	default:
		return false
	}
}
