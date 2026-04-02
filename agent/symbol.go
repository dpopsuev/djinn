// symbol.go — SymbolProvider port + named types for pre-edit impact awareness (TSK-645).
//
// Port defined at consumer (agent/) per DIP. Providers live in separate files:
// RegexProvider (regex_provider.go), ASTProvider (ast_provider.go),
// LSPProvider (lsp_provider.go).
package agent

import (
	"context"
	"errors"
	"unicode"
)

// ErrUnsupportedLanguage is returned when a provider cannot handle the file's language.
var ErrUnsupportedLanguage = errors.New("unsupported language")

// ErrNotConnected is returned when an LSP-based provider has no active connection.
var ErrNotConnected = errors.New("not connected")

// SymbolProvider resolves symbols and their references.
// Port defined at consumer (agent/) per DIP.
type SymbolProvider interface {
	Symbols(ctx context.Context, file string) ([]Symbol, error)
	References(ctx context.Context, file, symbol string) ([]Reference, error)
}

// SymbolKind classifies a symbol declaration.
type SymbolKind string

const (
	SymbolFunc      SymbolKind = "func"
	SymbolType      SymbolKind = "type"
	SymbolVar       SymbolKind = "var"
	SymbolConst     SymbolKind = "const"
	SymbolInterface SymbolKind = "interface"
	SymbolMethod    SymbolKind = "method"
)

// Symbol is a single declaration in a file.
type Symbol struct {
	Name     string
	Kind     SymbolKind
	Line     int
	Exported bool
}

// Reference is a location where a symbol is used.
type Reference struct {
	File string
	Line int
}

// SymbolGraph is the pre-edit impact table for a file.
type SymbolGraph struct {
	File    string
	Symbols []SymbolEntry
}

// SymbolEntry pairs a symbol with its internal and external callers.
type SymbolEntry struct {
	Symbol   Symbol
	Internal []Reference // callers in same package
	External []Reference // callers outside package
}

// isExportedGo returns true if the first rune is uppercase (Go convention).
func isExportedGo(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}
