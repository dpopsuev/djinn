// tier4_markers.go — Tier 4 heuristic: TODO/FIXME/HACK + magic numbers (TSK-464).
//
// Scans diff additions for deferred work markers and unexplained numeric literals.
package review

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// MarkersHeuristic detects TODO/FIXME/HACK comments and magic numbers.
type MarkersHeuristic struct {
	MaxTodos        int
	MaxMagicNumbers int
}

// NewMarkersHeuristic creates a heuristic with the given thresholds.
func NewMarkersHeuristic(maxTodos, maxMagic int) *MarkersHeuristic {
	return &MarkersHeuristic{
		MaxTodos:        maxTodos,
		MaxMagicNumbers: maxMagic,
	}
}

func (h *MarkersHeuristic) Name() string { return "tier4_markers" }

func (h *MarkersHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	var signals []Signal

	// Count TODOs in added/changed file names isn't useful —
	// we'd need the actual diff content. For now, scan the DiffSnapshot detail.
	// In practice, the agent's tool output contains the diff text.
	// This heuristic works on the file list as a proxy.

	if h.MaxTodos > 0 {
		todos := countMarkers(diff)
		signals = append(signals, Signal{
			Metric:    "todos_introduced",
			Value:     float64(todos),
			Threshold: float64(h.MaxTodos),
			Exceeded:  todos >= h.MaxTodos,
			Detail:    fmt.Sprintf("%d TODO/FIXME/HACK markers", todos),
		})
	}

	if h.MaxMagicNumbers > 0 {
		magics := countMagicNumbers(diff)
		signals = append(signals, Signal{
			Metric:    "magic_numbers",
			Value:     float64(magics),
			Threshold: float64(h.MaxMagicNumbers),
			Exceeded:  magics >= h.MaxMagicNumbers,
			Detail:    fmt.Sprintf("%d magic numbers", magics),
		})
	}

	return signals, nil
}

// Todo/fixme markers to scan for.
var todoMarkers = []string{"TODO", "FIXME", "HACK", "XXX", "NOCOMMIT"}

// countMarkers counts deferred-work markers in file content.
func countMarkers(diff *DiffSnapshot) int {
	count := 0
	allFiles := append(diff.ChangedFiles, diff.AddedFiles...) //nolint:gocritic // append to copy is intentional
	for _, f := range allFiles {
		if !isSourceFile(f) {
			continue
		}
		content := readFileContent(diff.WorkDir, f)
		for _, marker := range todoMarkers {
			count += strings.Count(content, marker)
		}
	}
	return count
}

// Magic number exclusions — common, well-understood values.
var commonNumbers = map[string]bool{
	"0": true, "1": true, "2": true, "-1": true,
	"100": true, "200": true, "201": true, "204": true, // HTTP status
	"400": true, "401": true, "403": true, "404": true, "500": true,
	"1024": true, "4096": true, "8192": true, // buffer sizes
	"0o600": true, "0o644": true, "0o700": true, "0o755": true, // permissions
	"0.0": true, "1.0": true, "0.5": true,
}

// countMagicNumbers counts unexplained numeric literals in source files.
func countMagicNumbers(diff *DiffSnapshot) int {
	count := 0
	for _, f := range diff.AddedFiles {
		if !isSourceFile(f) {
			continue
		}
		content := readFileContent(diff.WorkDir, f)
		count += scanMagicNumbers(content)
	}
	return count
}

func scanMagicNumbers(content string) int {
	count := 0
	words := strings.Fields(content)
	for _, w := range words {
		w = strings.Trim(w, "(),;:{}")
		if commonNumbers[w] {
			continue
		}
		// Check if it's a numeric literal.
		if _, err := strconv.ParseFloat(w, 64); err == nil { //nolint:mnd // 64-bit float parsing
			count++
		}
		if _, err := strconv.ParseInt(w, 0, 64); err == nil { //nolint:mnd // 64-bit int parsing
			if !commonNumbers[w] {
				count++
			}
		}
	}
	return count
}

func readFileContent(workDir, file string) string {
	if workDir == "" {
		return ""
	}
	data, err := readFileBytes(workDir, file)
	if err != nil {
		return ""
	}
	return string(data)
}
