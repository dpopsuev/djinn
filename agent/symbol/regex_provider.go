// regex_provider.go — Regex-based SymbolProvider (TSK-646).
//
// Day 1a: works for any language with simple declaration patterns.
// Go patterns: func, type, var, const at top-level.
// References: grep for symbol name across the workspace directory.
package symbol

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RegexProvider extracts symbols using regex patterns per language.
type RegexProvider struct{}

// languagePatterns maps file extensions to declaration regexes.
// Each regex must have a named group "name" for the symbol name.
var languagePatterns = map[string][]struct {
	re   *regexp.Regexp
	kind SymbolKind
}{
	".go": {
		{re: regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(?P<name>\w+)`), kind: SymbolFunc},
		{re: regexp.MustCompile(`^type\s+(?P<name>\w+)\s+interface\b`), kind: SymbolInterface},
		{re: regexp.MustCompile(`^type\s+(?P<name>\w+)`), kind: SymbolType},
		{re: regexp.MustCompile(`^var\s+(?P<name>\w+)`), kind: SymbolVar},
		{re: regexp.MustCompile(`^const\s+(?P<name>\w+)`), kind: SymbolConst},
	},
	".py": {
		{re: regexp.MustCompile(`^def\s+(?P<name>\w+)`), kind: SymbolFunc},
		{re: regexp.MustCompile(`^class\s+(?P<name>\w+)`), kind: SymbolType},
	},
	".js": {
		{re: regexp.MustCompile(`^(?:export\s+)?function\s+(?P<name>\w+)`), kind: SymbolFunc},
		{re: regexp.MustCompile(`^(?:export\s+)?class\s+(?P<name>\w+)`), kind: SymbolType},
		{re: regexp.MustCompile(`^(?:export\s+)?const\s+(?P<name>\w+)`), kind: SymbolConst},
		{re: regexp.MustCompile(`^(?:export\s+)?let\s+(?P<name>\w+)`), kind: SymbolVar},
	},
	".ts": {
		{re: regexp.MustCompile(`^(?:export\s+)?function\s+(?P<name>\w+)`), kind: SymbolFunc},
		{re: regexp.MustCompile(`^(?:export\s+)?class\s+(?P<name>\w+)`), kind: SymbolType},
		{re: regexp.MustCompile(`^(?:export\s+)?interface\s+(?P<name>\w+)`), kind: SymbolInterface},
		{re: regexp.MustCompile(`^(?:export\s+)?const\s+(?P<name>\w+)`), kind: SymbolConst},
		{re: regexp.MustCompile(`^(?:export\s+)?let\s+(?P<name>\w+)`), kind: SymbolVar},
	},
}

// Symbols extracts declarations from a file using regex patterns.
func (r *RegexProvider) Symbols(_ context.Context, file string) ([]Symbol, error) {
	ext := filepath.Ext(file)
	patterns, ok := languagePatterns[ext]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedLanguage, ext)
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", file, err)
	}
	defer f.Close()

	var symbols []Symbol
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, p := range patterns {
			m := p.re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			nameIdx := p.re.SubexpIndex("name")
			if nameIdx < 0 || nameIdx >= len(m) {
				continue
			}
			name := m[nameIdx]

			// For Go: detect method vs function from receiver.
			kind := p.kind
			if ext == ".go" && kind == SymbolFunc && strings.Contains(line, ") "+name) {
				kind = SymbolMethod
			}

			exported := isExportedGo(name)
			if ext != ".go" {
				// For non-Go languages, default exported = true
				exported = true
			}

			symbols = append(symbols, Symbol{
				Name:     name,
				Kind:     kind,
				Line:     lineNum,
				Exported: exported,
			})
			break // first match wins per line
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", file, err)
	}

	return symbols, nil
}

// References finds callers of a symbol by scanning files in the workspace.
// The workspace is derived as the parent directory of the file.
// The definition file itself is excluded from results.
func (r *RegexProvider) References(_ context.Context, file, symbol string) ([]Reference, error) {
	workspace := filepath.Dir(file)
	absFile, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	// Build a regex that matches the symbol as a whole word.
	wordRe, err := regexp.Compile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	if err != nil {
		return nil, fmt.Errorf("compile symbol regex: %w", err)
	}

	var refs []Reference
	err = filepath.WalkDir(workspace, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable dirs
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan known source files.
		ext := filepath.Ext(path)
		if _, ok := languagePatterns[ext]; !ok {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		// Skip the definition file itself.
		if absPath == absFile {
			return nil
		}

		lineRefs, err := scanFileForSymbol(absPath, wordRe)
		if err != nil {
			return nil // skip unreadable files
		}
		refs = append(refs, lineRefs...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", workspace, err)
	}

	return refs, nil
}

// scanFileForSymbol reads a file line-by-line and returns references where the regex matches.
func scanFileForSymbol(path string, re *regexp.Regexp) ([]Reference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var refs []Reference
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if re.MatchString(scanner.Text()) {
			refs = append(refs, Reference{
				File: path,
				Line: lineNum,
			})
		}
	}
	return refs, scanner.Err()
}
