// real.go — RealLector: production symbol cache wrapping Oculus.
//
// Lazy indexing: indexes on first OnFileRead, not full codebase scan.
// Content-addressed: cache key includes file content hash (from Canon).
// Delegates to Oculus ClassAnalyzer for symbol extraction.
//
// GOL-152, TSK-1113
package lector

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/sahilm/fuzzy"
)

var _ Lector = (*RealLector)(nil)

// RealLector indexes files and symbols via Oculus. Lazy — only indexes
// files that agents actually read. Thread-safe.
type RealLector struct {
	mu      sync.RWMutex
	files   map[string]FileEntry // path → file metadata
	symbols map[string][]Symbol  // file → symbols in that file
	log     *slog.Logger
}

// NewRealLector creates a production Lector.
// Uses go/parser for Go files (Day 1). Oculus integration (Day 2) adds
// multi-language support via tree-sitter and LSP.
func NewRealLector(_ string, log *slog.Logger) *RealLector {
	if log == nil {
		log = telemetry.Nop()
	}
	return &RealLector{
		files:   make(map[string]FileEntry),
		symbols: make(map[string][]Symbol),
		log:     log,
	}
}

// --- Index (read) ---

func (l *RealLector) FileInfo(path string) (FileEntry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	f, ok := l.files[path]
	return f, ok
}

func (l *RealLector) Symbols(scope string) []Symbol {
	l.mu.RLock()
	defer l.mu.RUnlock()
	// Scope = package name. Collect from all files in that package.
	var out []Symbol
	for _, syms := range l.symbols {
		for _, s := range syms {
			if s.Package == scope {
				out = append(out, s)
			}
		}
	}
	return out
}

func (l *RealLector) SymbolsForFile(file string) []Symbol {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Symbol, len(l.symbols[file]))
	copy(out, l.symbols[file])
	return out
}

func (l *RealLector) Imports(pkg string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, f := range l.files {
		if f.Package == pkg && len(f.Imports) > 0 {
			out := make([]string, len(f.Imports))
			copy(out, f.Imports)
			return out
		}
	}
	return nil
}

func (l *RealLector) Dependents(pkg string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var deps []string
	for _, f := range l.files {
		for _, imp := range f.Imports {
			if imp == pkg {
				deps = append(deps, f.Path)
				break
			}
		}
	}
	return deps
}

func (l *RealLector) FuzzyFiles(query string) []FileEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Build searchable list.
	paths := make([]string, 0, len(l.files))
	entries := make([]FileEntry, 0, len(l.files))
	for _, f := range l.files {
		paths = append(paths, f.Path)
		entries = append(entries, f)
	}

	matches := fuzzy.Find(query, paths)
	results := make([]FileEntry, 0, len(matches))
	for _, m := range matches {
		results = append(results, entries[m.Index])
	}
	return results
}

func (l *RealLector) FuzzySymbols(query string) []Symbol {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Flatten all symbols into searchable list.
	var names []string
	var allSyms []Symbol
	for _, syms := range l.symbols {
		for _, s := range syms {
			names = append(names, s.Name)
			allSyms = append(allSyms, s)
		}
	}

	matches := fuzzy.Find(query, names)
	results := make([]Symbol, 0, len(matches))
	for _, m := range matches {
		results = append(results, allSyms[m.Index])
	}
	return results
}

// --- Observer (write) ---

func (l *RealLector) OnFileRead(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Index file metadata.
	info, err := os.Stat(path)
	if err != nil {
		l.log.WarnContext(context.Background(), "lector: file not found on read",
			slog.String(telemetry.KeyPath, path),
		)
		// Still record the path for fuzzy search.
		l.files[path] = FileEntry{Path: path}
		return
	}

	lang := detectLanguage(path)
	fe := FileEntry{
		Path:         path,
		Language:     lang,
		Size:         info.Size(),
		LastModified: info.ModTime().Unix(),
	}
	l.files[path] = fe

	// Extract symbols via Oculus (only for supported languages).
	if isCodeFile(lang) {
		l.indexSymbolsLocked(path, fe)
	}
}

func (l *RealLector) OnFileWrite(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Invalidate cached symbols.
	delete(l.symbols, path)

	// Re-index metadata.
	info, err := os.Stat(path)
	if err != nil {
		l.files[path] = FileEntry{Path: path}
		return
	}

	lang := detectLanguage(path)
	fe := FileEntry{
		Path:         path,
		Language:     lang,
		Size:         info.Size(),
		LastModified: info.ModTime().Unix(),
	}
	l.files[path] = fe

	if isCodeFile(lang) {
		l.indexSymbolsLocked(path, fe)
	}
}

func (l *RealLector) OnFileDelete(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.files, path)
	delete(l.symbols, path)
}

// --- Internal ---

// indexSymbolsLocked extracts symbols via go/parser. Caller must hold l.mu.
// Day 1: Go only. Day 2: Oculus FallbackAnalyzer adds 9 languages via tree-sitter.
func (l *RealLector) indexSymbolsLocked(path string, fe FileEntry) {
	if fe.Language != langGo {
		return // Only Go for now.
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		l.log.DebugContext(context.Background(), "lector: parse failed",
			slog.String(telemetry.KeyPath, path),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return
	}

	pkgName := ""
	if f.Name != nil {
		pkgName = f.Name.Name
	}

	var syms []Symbol
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			kind := "func"
			if d.Recv != nil {
				kind = "method"
			}
			syms = append(syms, Symbol{
				Name:     d.Name.Name,
				Kind:     kind,
				Package:  pkgName,
				File:     path,
				Line:     fset.Position(d.Pos()).Line,
				Exported: ast.IsExported(d.Name.Name),
			})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					kind := "type"
					if _, ok := s.Type.(*ast.InterfaceType); ok {
						kind = "interface"
					}
					syms = append(syms, Symbol{
						Name:     s.Name.Name,
						Kind:     kind,
						Package:  pkgName,
						File:     path,
						Line:     fset.Position(s.Pos()).Line,
						Exported: ast.IsExported(s.Name.Name),
					})
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						syms = append(syms, Symbol{
							Name:     name.Name,
							Kind:     kind,
							Package:  pkgName,
							File:     path,
							Line:     fset.Position(name.Pos()).Line,
							Exported: ast.IsExported(name.Name),
						})
					}
				}
			}
		}
	}

	if len(syms) > 0 {
		l.symbols[path] = syms
		l.log.DebugContext(context.Background(), "lector: indexed",
			slog.String(telemetry.KeyPath, path),
			slog.Int(telemetry.KeyCount, len(syms)),
		)
	}

	// Update package name.
	if pkgName != "" {
		fe.Package = pkgName
		l.files[path] = fe
	}
}

const langGo = "go"

// detectLanguage returns the language based on file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return langGo
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".kt":
		return "kotlin"
	case ".cs":
		return "csharp"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp":
		return "cpp"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	case ".sh", ".bash":
		return "shell"
	default:
		return "text"
	}
}

// isCodeFile returns true if the language has symbols (programming language, not config).
func isCodeFile(lang string) bool {
	switch lang {
	case langGo, "python", "typescript", "javascript", "rust", "java", "kotlin", "csharp", "c", "cpp":
		return true
	default:
		return false
	}
}
