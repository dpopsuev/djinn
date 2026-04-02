// ast_provider.go — AST-based SymbolProvider for Go files (TSK-647).
//
// Uses go/ast + go/parser for precise declaration extraction.
// More accurate than regex — handles methods, receivers, and interface declarations.
// Single-file only; cross-file references fall back to RegexProvider.
package agent

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

const goFileExt = ".go"

// ASTProvider extracts symbols from Go files using go/ast.
type ASTProvider struct {
	// fallback is used for References (AST is single-file only).
	fallback *RegexProvider
}

// NewASTProvider creates an ASTProvider with a regex fallback for references.
func NewASTProvider() *ASTProvider {
	return &ASTProvider{fallback: &RegexProvider{}}
}

// Symbols parses a Go file and returns all top-level declarations.
func (a *ASTProvider) Symbols(_ context.Context, file string) ([]Symbol, error) {
	ext := filepath.Ext(file)
	if ext != goFileExt {
		return nil, fmt.Errorf("%w: %s (AST provider supports Go only)", ErrUnsupportedLanguage, ext)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}

	var symbols []Symbol

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			kind := SymbolFunc
			if d.Recv != nil {
				kind = SymbolMethod
			}
			symbols = append(symbols, Symbol{
				Name:     d.Name.Name,
				Kind:     kind,
				Line:     fset.Position(d.Pos()).Line,
				Exported: d.Name.IsExported(),
			})

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					kind := SymbolType
					if _, ok := s.Type.(*ast.InterfaceType); ok {
						kind = SymbolInterface
					}
					symbols = append(symbols, Symbol{
						Name:     s.Name.Name,
						Kind:     kind,
						Line:     fset.Position(s.Pos()).Line,
						Exported: s.Name.IsExported(),
					})

				case *ast.ValueSpec:
					kind := SymbolVar
					if d.Tok == token.CONST {
						kind = SymbolConst
					}
					for _, name := range s.Names {
						// Skip blank identifiers.
						if name.Name == "_" {
							continue
						}
						symbols = append(symbols, Symbol{
							Name:     name.Name,
							Kind:     kind,
							Line:     fset.Position(name.Pos()).Line,
							Exported: name.IsExported(),
						})
					}
				}
			}
		}
	}

	return symbols, nil
}

// References delegates to the regex fallback — AST is single-file only.
func (a *ASTProvider) References(ctx context.Context, file, symbol string) ([]Reference, error) {
	return a.fallback.References(ctx, file, symbol)
}
