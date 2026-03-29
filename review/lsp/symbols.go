// symbols.go — LSP textDocument/documentSymbol types and request helpers.
package lsp

import "encoding/json"

// SymbolKind mirrors the LSP specification SymbolKind enum.
type SymbolKind int

// LSP SymbolKind constants.
const (
	SymbolFile          SymbolKind = 1
	SymbolModule        SymbolKind = 2
	SymbolNamespace     SymbolKind = 3
	SymbolPackage       SymbolKind = 4
	SymbolClass         SymbolKind = 5
	SymbolMethod        SymbolKind = 6
	SymbolProperty      SymbolKind = 7
	SymbolField         SymbolKind = 8
	SymbolConstructor   SymbolKind = 9
	SymbolEnum          SymbolKind = 10
	SymbolInterface     SymbolKind = 11
	SymbolFunction      SymbolKind = 12
	SymbolVariable      SymbolKind = 13
	SymbolConstant      SymbolKind = 14
	SymbolString        SymbolKind = 15
	SymbolNumber        SymbolKind = 16
	SymbolBoolean       SymbolKind = 17
	SymbolArray         SymbolKind = 18
	SymbolObject        SymbolKind = 19
	SymbolKey           SymbolKind = 20
	SymbolNull          SymbolKind = 21
	SymbolEnumMember    SymbolKind = 22
	SymbolStruct        SymbolKind = 23
	SymbolEvent         SymbolKind = 24
	SymbolOperator      SymbolKind = 25
	SymbolTypeParameter SymbolKind = 26
)

// String returns the LSP name for the symbol kind.
func (sk SymbolKind) String() string {
	names := map[SymbolKind]string{
		SymbolFile: "File", SymbolModule: "Module", SymbolNamespace: "Namespace",
		SymbolPackage: "Package", SymbolClass: "Class", SymbolMethod: "Method",
		SymbolProperty: "Property", SymbolField: "Field", SymbolConstructor: "Constructor",
		SymbolEnum: "Enum", SymbolInterface: "Interface", SymbolFunction: "Function",
		SymbolVariable: "Variable", SymbolConstant: "Constant", SymbolString: "String",
		SymbolStruct: "Struct", SymbolTypeParameter: "TypeParameter",
	}
	if name, ok := names[sk]; ok {
		return name
	}
	return "Unknown"
}

// IsExportable returns true for symbol kinds that can be exported/public.
func (sk SymbolKind) IsExportable() bool {
	switch sk {
	case SymbolFunction, SymbolMethod, SymbolClass, SymbolInterface,
		SymbolStruct, SymbolEnum, SymbolConstant, SymbolVariable,
		SymbolProperty, SymbolField, SymbolTypeParameter:
		return true
	default:
		return false
	}
}

// Position in an LSP document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range in an LSP document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// TextDocumentIdentifier identifies a document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// DocumentSymbolParams is the request payload for textDocument/documentSymbol.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentSymbol represents a symbol in a document (hierarchical).
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"` // signature, type info
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// RequestDocumentSymbols calls textDocument/documentSymbol on the given URI.
func RequestDocumentSymbols(c *Client, uri string) ([]DocumentSymbol, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	result, err := c.Request("textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}
	var symbols []DocumentSymbol
	if err := json.Unmarshal(result, &symbols); err != nil {
		return nil, err
	}
	return symbols, nil
}

// FlattenSymbols recursively flattens hierarchical document symbols.
func FlattenSymbols(symbols []DocumentSymbol) []DocumentSymbol {
	var flat []DocumentSymbol
	var walk func(syms []DocumentSymbol)
	walk = func(syms []DocumentSymbol) {
		for i := range syms {
			flat = append(flat, syms[i])
			if len(syms[i].Children) > 0 {
				walk(syms[i].Children)
			}
		}
	}
	walk(symbols)
	return flat
}
