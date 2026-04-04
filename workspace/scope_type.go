// scope_type.go — orthogonal scope type taxonomy (SPC-94).
//
// ScopeType is independent of Level (position in hierarchy).
// Level = WHERE you are (General/Ecosystem/System).
// ScopeType = WHAT kind of scope it is (Global/Ecosystem/System/Prototype/Operations).
//
// Any position can overlay with Operations (fire-and-forget ops tasks).
package workspace

import (
	"os"
	"path/filepath"
)

// ScopeType classifies a scope's behavior and mount semantics.
type ScopeType string

const (
	// ScopeGlobal is the root scope — read everything, write nothing.
	ScopeGlobal ScopeType = "global"

	// ScopeEcosystem is a multi-repo workspace (go.work).
	ScopeEcosystem ScopeType = "ecosystem"

	// ScopeSystem is a single repo (.git).
	ScopeSystem ScopeType = "system"

	// ScopePrototype is an experimental branch — sandboxed writes.
	ScopePrototype ScopeType = "prototype"

	// ScopeOperations is a fire-and-forget ops overlay — ephemeral mounts.
	ScopeOperations ScopeType = "operations"
)

// InferScopeType examines the filesystem at path and returns the best-fit
// scope type:
//   - If go.work exists, it's an ecosystem (multi-repo workspace).
//   - If .git exists, it's a system (single repo).
//   - Otherwise, global (read-only root).
func InferScopeType(path string) ScopeType {
	// Check for go.work (ecosystem — multi-repo workspace).
	if _, err := os.Stat(filepath.Join(path, "go.work")); err == nil {
		return ScopeEcosystem
	}

	// Check for .git (system — single repo).
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return ScopeSystem
	}

	return ScopeGlobal
}

// Valid returns true if the scope type is one of the known values.
func (st ScopeType) Valid() bool {
	switch st {
	case ScopeGlobal, ScopeEcosystem, ScopeSystem, ScopePrototype, ScopeOperations:
		return true
	default:
		return false
	}
}

// String returns the string representation of the scope type.
func (st ScopeType) String() string {
	return string(st)
}
