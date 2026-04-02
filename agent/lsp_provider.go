// lsp_provider.go — LSP-based SymbolProvider (TSK-648).
//
// Wraps review/lsp.Client for textDocument/documentSymbol.
// References() is Day 2 — stub returns ErrNotConnected for now.
package agent

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/dpopsuev/djinn/review/lsp"
)

// LSPProvider wraps an LSP client for symbol resolution.
type LSPProvider struct {
	client *lsp.Client
}

// NewLSPProvider creates an LSPProvider backed by the given LSP client.
func NewLSPProvider(client *lsp.Client) *LSPProvider {
	return &LSPProvider{client: client}
}

// Symbols calls textDocument/documentSymbol via the LSP client.
func (l *LSPProvider) Symbols(_ context.Context, file string) ([]Symbol, error) {
	if l.client == nil {
		return nil, ErrNotConnected
	}

	absPath, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}
	uri := "file://" + url.PathEscape(absPath)

	docSymbols, err := lsp.RequestDocumentSymbols(l.client, uri)
	if err != nil {
		return nil, fmt.Errorf("lsp documentSymbol: %w", err)
	}

	flat := lsp.FlattenSymbols(docSymbols)
	symbols := make([]Symbol, 0, len(flat))
	for i := range flat {
		kind := mapLSPKind(flat[i].Kind)
		name := flat[i].Name
		symbols = append(symbols, Symbol{
			Name:     name,
			Kind:     kind,
			Line:     flat[i].Range.Start.Line + 1, // LSP is 0-based
			Exported: isExportedGo(name),
		})
	}

	return symbols, nil
}

// References is Day 2 — LSP textDocument/references not yet implemented.
func (l *LSPProvider) References(_ context.Context, _, _ string) ([]Reference, error) {
	return nil, ErrNotConnected
}

// mapLSPKind converts an LSP SymbolKind to our SymbolKind.
func mapLSPKind(k lsp.SymbolKind) SymbolKind {
	switch k {
	case lsp.SymbolFunction:
		return SymbolFunc
	case lsp.SymbolMethod:
		return SymbolMethod
	case lsp.SymbolClass, lsp.SymbolStruct:
		return SymbolType
	case lsp.SymbolInterface:
		return SymbolInterface
	case lsp.SymbolVariable:
		return SymbolVar
	case lsp.SymbolConstant:
		return SymbolConst
	default:
		return SymbolType // safe fallback
	}
}
