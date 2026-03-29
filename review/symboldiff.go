// symboldiff.go — Symbol diff engine using LSP documentSymbol (TSK-453).
//
// Compares symbol declarations before/after file changes.
// Works with any language that has an LSP server installed.
package review

import (
	"github.com/dpopsuev/djinn/review/lsp"
)

// SymbolInfo captures a single symbol extracted via LSP.
type SymbolInfo struct {
	Name     string         `json:"name"`
	Kind     lsp.SymbolKind `json:"kind"`
	Detail   string         `json:"detail"`   // signature or type info from LSP
	Exported bool           `json:"exported"` // language-specific visibility
	Children int            `json:"children"` // method/field count for interfaces/structs
}

// SymbolDiff captures the difference between two symbol sets.
type SymbolDiff struct {
	Added    []SymbolInfo   `json:"added"`
	Removed  []SymbolInfo   `json:"removed"`
	Modified []SymbolChange `json:"modified"`
}

// SymbolChange captures a modification to an existing symbol.
type SymbolChange struct {
	Name           string         `json:"name"`
	Kind           lsp.SymbolKind `json:"kind"`
	Exported       bool           `json:"exported"`
	DetailBefore   string         `json:"detail_before"`
	DetailAfter    string         `json:"detail_after"`
	ChildrenBefore int            `json:"children_before"`
	ChildrenAfter  int            `json:"children_after"`
}

// DiffSymbolSets compares two symbol sets and returns the diff.
func DiffSymbolSets(before, after []SymbolInfo) *SymbolDiff {
	beforeMap := make(map[string]SymbolInfo, len(before))
	for _, s := range before {
		beforeMap[s.Name] = s
	}
	afterMap := make(map[string]SymbolInfo, len(after))
	for _, s := range after {
		afterMap[s.Name] = s
	}

	diff := &SymbolDiff{}

	// Added: in after but not in before.
	for _, s := range after {
		if _, found := beforeMap[s.Name]; !found {
			diff.Added = append(diff.Added, s)
		}
	}

	// Removed: in before but not in after.
	for _, s := range before {
		if _, found := afterMap[s.Name]; !found {
			diff.Removed = append(diff.Removed, s)
		}
	}

	// Modified: in both but changed.
	for _, a := range after {
		b, found := beforeMap[a.Name]
		if !found {
			continue
		}
		if b.Detail != a.Detail || b.Children != a.Children || b.Kind != a.Kind {
			diff.Modified = append(diff.Modified, SymbolChange{
				Name:           a.Name,
				Kind:           a.Kind,
				Exported:       a.Exported,
				DetailBefore:   b.Detail,
				DetailAfter:    a.Detail,
				ChildrenBefore: b.Children,
				ChildrenAfter:  a.Children,
			})
		}
	}

	return diff
}

// SymbolsFromLSP converts LSP DocumentSymbols to SymbolInfo list.
func SymbolsFromLSP(symbols []lsp.DocumentSymbol) []SymbolInfo {
	result := make([]SymbolInfo, 0, len(symbols))
	for i := range symbols {
		s := &symbols[i]
		result = append(result, SymbolInfo{
			Name:     s.Name,
			Kind:     s.Kind,
			Detail:   s.Detail,
			Exported: isExported(s.Name, s.Kind),
			Children: len(s.Children),
		})
	}
	return result
}

// isExported applies language-agnostic export heuristics.
// Go: uppercase first letter = exported, lowercase = unexported.
// Python: underscore prefix = private.
// Other: assume exported (TS export, Rust pub detected by LSP detail).
func isExported(name string, kind lsp.SymbolKind) bool {
	if !kind.IsExportable() {
		return false
	}
	if name == "" {
		return false
	}
	first := name[0]
	// Go convention: uppercase = exported, lowercase = unexported.
	if first >= 'a' && first <= 'z' {
		return false
	}
	// Python convention: underscore = private.
	if first == '_' {
		return false
	}
	return true
}
