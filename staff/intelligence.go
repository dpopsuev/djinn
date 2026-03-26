// intelligence.go — Pre-flight intelligence gathering for the GenSec.
// Scans the working directory for structural information that helps
// the secretary make better delegation decisions.
package staff

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// IntelBriefing contains pre-flight intelligence about the workspace.
type IntelBriefing struct {
	ExistingPackages []string // Go packages found in the workspace
	OpenTopics       []string // placeholder for discourse topics
	FailingTests     int      // count of *_test.go files (proxy for test surface)
	Recommendations  []string // actionable advice for the secretary
}

// GatherIntelligence scans the working directory and returns a structured
// briefing. It walks the file tree to discover Go packages and test files.
// Context is respected for cancellation.
func GatherIntelligence(ctx context.Context, workDir string) *IntelBriefing {
	b := &IntelBriefing{}

	if workDir == "" {
		b.Recommendations = append(b.Recommendations, "no working directory specified")
		return b
	}

	pkgSet := make(map[string]bool)
	testCount := 0

	_ = filepath.WalkDir(workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}

		// Respect context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip hidden directories and vendor.
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only care about Go files.
		if !strings.HasSuffix(name, ".go") {
			return nil
		}

		// Track packages (directory-level).
		dir := filepath.Dir(path)
		rel, relErr := filepath.Rel(workDir, dir)
		if relErr == nil && rel != "" {
			pkgSet[rel] = true
		}

		// Count test files.
		if strings.HasSuffix(name, "_test.go") {
			testCount++
		}

		return nil
	})

	// Convert set to sorted slice.
	for pkg := range pkgSet {
		b.ExistingPackages = append(b.ExistingPackages, pkg)
	}

	b.FailingTests = testCount

	// Generate recommendations.
	if len(b.ExistingPackages) > 0 {
		b.Recommendations = append(b.Recommendations,
			"existing packages found — extend, don't rewrite")
	}
	if testCount == 0 {
		b.Recommendations = append(b.Recommendations,
			"no test files detected — add tests")
	}

	return b
}
