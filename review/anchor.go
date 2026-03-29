// anchor.go — Scope Anchor + drift detection (TSK-438).
//
// Captures the operator's original request and measures scope drift:
// are the changed files within the expected packages?
package review

import (
	"path/filepath"
	"strings"
)

// DriftVerdict summarizes whether changes are on scope.
type DriftVerdict string

const (
	VerdictOnScope DriftVerdict = "ON_SCOPE"
	VerdictDrift   DriftVerdict = "DRIFT_DETECTED"
	VerdictUnknown DriftVerdict = "SCOPE_UNKNOWN"
)

// ScopeAnchor captures the original request and expected scope.
type ScopeAnchor struct {
	OriginalRequest  string
	ExpectedPackages []string // directories expected to be changed
}

// NewScopeAnchor creates an anchor by extracting package names from the request.
func NewScopeAnchor(request string) *ScopeAnchor {
	return &ScopeAnchor{
		OriginalRequest:  request,
		ExpectedPackages: extractPackages(request),
	}
}

// CheckDrift compares changed files against expected packages.
// Returns the verdict and list of files outside expected scope.
func (sa *ScopeAnchor) CheckDrift(changedFiles []string) (verdict DriftVerdict, driftedFiles []string) {
	if len(sa.ExpectedPackages) == 0 {
		return VerdictUnknown, nil
	}

	var drifted []string
	for _, f := range changedFiles {
		if !sa.isInScope(f) {
			drifted = append(drifted, f)
		}
	}

	if len(drifted) == 0 {
		return VerdictOnScope, nil
	}
	return VerdictDrift, drifted
}

func (sa *ScopeAnchor) isInScope(file string) bool {
	dir := filepath.Dir(file)
	for _, pkg := range sa.ExpectedPackages {
		if dir == pkg || strings.HasPrefix(dir, pkg+"/") || strings.HasPrefix(dir, pkg+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// extractPackages pulls directory-like tokens from the request text.
// Extracts the directory part of any path containing "/".
func extractPackages(request string) []string {
	words := strings.Fields(request)
	var pkgs []string
	seen := make(map[string]bool)
	for _, w := range words {
		// Strip punctuation.
		w = strings.Trim(w, ".,;:!?\"'`()[]{}") //nolint:gocritic // trimming multiple chars is intentional
		if !strings.Contains(w, "/") || strings.HasPrefix(w, "http") {
			continue
		}
		// Use directory part — "auth/handler.go" → "auth"
		dir := filepath.Dir(w)
		if dir == "." {
			continue
		}
		if !seen[dir] {
			seen[dir] = true
			pkgs = append(pkgs, dir)
		}
	}
	return pkgs
}
