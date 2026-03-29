// tier1_deps.go — Tier 1 heuristic: dependency + config file change detector (TSK-451).
package review

import (
	"context"
	"path/filepath"
	"strings"
)

// Known dependency file names — supply chain changes warrant immediate review.
var dependencyFiles = map[string]bool{
	"go.mod":            true,
	"go.sum":            true,
	"package.json":      true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"Cargo.toml":        true,
	"Cargo.lock":        true,
	"requirements.txt":  true,
	"pyproject.toml":    true,
	"poetry.lock":       true,
	"pom.xml":           true,
	"build.gradle":      true,
	"build.gradle.kts":  true,
}

// Config file patterns — changes with outsized blast radius.
var configExtensions = map[string]bool{
	".yaml": true,
	".yml":  true,
	".toml": true,
	".json": true,
	".env":  true,
}

// DepsHeuristic detects dependency and config file changes.
type DepsHeuristic struct {
	OnNewDependency bool
}

// NewDepsHeuristic creates a heuristic with the given config.
func NewDepsHeuristic(cfg Config) *DepsHeuristic {
	return &DepsHeuristic{
		OnNewDependency: cfg.OnNewDependency,
	}
}

func (h *DepsHeuristic) Name() string { return "tier1_deps" }

func (h *DepsHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if !h.OnNewDependency {
		return nil, nil
	}

	var signals []Signal

	// Check all file lists for dependency files.
	allFiles := make([]string, 0, len(diff.ChangedFiles)+len(diff.AddedFiles)+len(diff.DeletedFiles))
	allFiles = append(allFiles, diff.ChangedFiles...)
	allFiles = append(allFiles, diff.AddedFiles...)
	allFiles = append(allFiles, diff.DeletedFiles...)

	var depFiles []string
	var cfgFiles []string

	for _, f := range allFiles {
		base := filepath.Base(f)
		if dependencyFiles[base] {
			depFiles = append(depFiles, f)
		} else if isConfigFile(f) {
			cfgFiles = append(cfgFiles, f)
		}
	}

	if len(depFiles) > 0 {
		signals = append(signals, Signal{
			Metric:    "dependency_changed",
			Value:     float64(len(depFiles)),
			Threshold: 1,
			Exceeded:  true,
			Detail:    "dependency files: " + strings.Join(depFiles, ", "),
		})
	}

	if len(cfgFiles) > 0 {
		signals = append(signals, Signal{
			Metric:    "config_changed",
			Value:     float64(len(cfgFiles)),
			Threshold: 1,
			Exceeded:  true,
			Detail:    "config files: " + strings.Join(cfgFiles, ", "),
		})
	}

	return signals, nil
}

func isConfigFile(path string) bool {
	ext := filepath.Ext(path)
	if !configExtensions[ext] {
		return false
	}
	// Only flag config files in config-like directories or at root.
	dir := filepath.Dir(path)
	return dir == "." || strings.Contains(dir, "config") || strings.HasPrefix(filepath.Base(path), ".")
}
