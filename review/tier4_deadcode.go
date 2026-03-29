// tier4_deadcode.go — Tier 4 heuristic: dead code + error handling ratio (TSK-465).
//
// Detects new unexported symbols with no references in the changed file set,
// and counts functions returning error vs total new functions.
package review

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeadCodeHeuristic checks for dead code and missing error handling.
type DeadCodeHeuristic struct {
	MaxDeadCode   int
	MinErrorRatio float64 // minimum ratio of error-returning funcs (default: 0.3)
}

// NewDeadCodeHeuristic creates a heuristic with the given thresholds.
func NewDeadCodeHeuristic(maxDead int, minErrorRatio float64) *DeadCodeHeuristic {
	return &DeadCodeHeuristic{
		MaxDeadCode:   maxDead,
		MinErrorRatio: minErrorRatio,
	}
}

func (h *DeadCodeHeuristic) Name() string { return "tier4_deadcode" }

func (h *DeadCodeHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if diff.WorkDir == "" {
		return nil, nil
	}

	var signals []Signal

	if h.MaxDeadCode > 0 {
		dead := h.detectDeadCode(diff)
		signals = append(signals, Signal{
			Metric:    "dead_code",
			Value:     float64(dead),
			Threshold: float64(h.MaxDeadCode),
			Exceeded:  dead >= h.MaxDeadCode,
			Detail:    fmt.Sprintf("%d potentially unreferenced symbols in new files", dead),
		})
	}

	if h.MinErrorRatio > 0 {
		totalFuncs, errorFuncs := h.countErrorHandling(diff)
		if totalFuncs > 0 {
			ratio := float64(errorFuncs) / float64(totalFuncs)
			signals = append(signals, Signal{
				Metric:    "error_handling_ratio",
				Value:     ratio,
				Threshold: h.MinErrorRatio,
				Exceeded:  ratio < h.MinErrorRatio,
				Detail:    fmt.Sprintf("%d/%d functions return error (%.0f%%)", errorFuncs, totalFuncs, ratio*100),
			})
		}
	}

	return signals, nil
}

// detectDeadCode finds unexported function declarations in new files that
// aren't referenced by any other file in the change set.
func (h *DeadCodeHeuristic) detectDeadCode(diff *DiffSnapshot) int {
	// Collect all content from changed + added files for reference scanning.
	allContent := make(map[string]string)
	allFiles := append(diff.ChangedFiles, diff.AddedFiles...) //nolint:gocritic // append to copy
	for _, f := range allFiles {
		if !isSourceFile(f) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(diff.WorkDir, f)) //nolint:gosec // trusted path
		if err != nil {
			continue
		}
		allContent[f] = string(data)
	}

	dead := 0
	// Only check new files for dead code — changed files might have existing callers.
	for _, f := range diff.AddedFiles {
		content, ok := allContent[f]
		if !ok {
			continue
		}
		funcNames := extractFuncNames(content)
		for _, name := range funcNames {
			if isExportedName(name) {
				continue // exported symbols may be used externally
			}
			if !isReferencedElsewhere(name, f, allContent) {
				dead++
			}
		}
	}
	return dead
}

// countErrorHandling counts functions and error-returning functions in new files.
func (h *DeadCodeHeuristic) countErrorHandling(diff *DiffSnapshot) (total, withError int) {
	for _, f := range diff.AddedFiles {
		if !isSourceFile(f) || !strings.HasSuffix(f, ".go") {
			continue // error return detection is Go-specific for now
		}
		data, err := os.ReadFile(filepath.Join(diff.WorkDir, f)) //nolint:gosec // trusted path
		if err != nil {
			continue
		}
		content := string(data)
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "func ") {
				total++
				if strings.Contains(trimmed, "error") {
					withError++
				}
			}
		}
	}
	return total, withError
}

// extractFuncNames pulls function/method names from source using simple heuristics.
func extractFuncNames(content string) []string {
	var names []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "func ") {
			continue
		}
		name := parseFuncName(trimmed)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// parseFuncName extracts the function name from a "func" declaration line.
func parseFuncName(line string) string {
	// "func Name(" or "func (r *T) Name("
	rest := strings.TrimPrefix(line, "func ")
	if strings.HasPrefix(rest, "(") {
		// Method: skip receiver.
		idx := strings.Index(rest, ") ")
		if idx < 0 {
			return ""
		}
		rest = rest[idx+2:]
	}
	paren := strings.Index(rest, "(")
	if paren < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:paren])
}

func isExportedName(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

func isReferencedElsewhere(name, sourceFile string, allContent map[string]string) bool {
	for f, content := range allContent {
		if f == sourceFile {
			continue
		}
		if strings.Contains(content, name) {
			return true
		}
	}
	return false
}

// readFileBytes reads a file from the working directory.
func readFileBytes(workDir, file string) ([]byte, error) {
	return os.ReadFile(filepath.Join(workDir, file)) //nolint:gosec // trusted path
}
