package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegexProvider_Symbols_GoFile(t *testing.T) {
	// Create a temp Go file with known declarations.
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	src := `package sample

func PublicFunc() {}
func privateFunc() {}

type MyStruct struct{}

func (m *MyStruct) Method() {}

type MyInterface interface {
	Do()
}

var GlobalVar = 42

const MaxRetries = 3
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	prov := &RegexProvider{}
	symbols, err := prov.Symbols(context.Background(), file)
	if err != nil {
		t.Fatalf("Symbols: %v", err)
	}

	// Expected: PublicFunc, privateFunc, MyStruct, Method, MyInterface, GlobalVar, MaxRetries
	if len(symbols) < 7 {
		t.Fatalf("expected >= 7 symbols, got %d: %+v", len(symbols), symbols)
	}

	// Check specific symbols.
	byName := make(map[string]Symbol)
	for _, s := range symbols {
		byName[s.Name] = s
	}

	assertSymbol(t, byName, "PublicFunc", SymbolFunc, true)
	assertSymbol(t, byName, "privateFunc", SymbolFunc, false)
	assertSymbol(t, byName, "MyStruct", SymbolType, true)
	assertSymbol(t, byName, "Method", SymbolMethod, true)
	assertSymbol(t, byName, "GlobalVar", SymbolVar, true)
	assertSymbol(t, byName, "MaxRetries", SymbolConst, true)
}

func TestRegexProvider_References_FindsCallers(t *testing.T) {
	dir := t.TempDir()

	// Definition file.
	defFile := filepath.Join(dir, "handler.go")
	defSrc := `package sample

func ValidateToken(token string) bool {
	return token != ""
}
`
	if err := os.WriteFile(defFile, []byte(defSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Caller file.
	callerFile := filepath.Join(dir, "middleware.go")
	callerSrc := `package sample

func Middleware() {
	ok := ValidateToken("abc")
	_ = ok
}
`
	if err := os.WriteFile(callerFile, []byte(callerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	prov := &RegexProvider{}
	refs, err := prov.References(context.Background(), defFile, "ValidateToken")
	if err != nil {
		t.Fatalf("References: %v", err)
	}

	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d: %+v", len(refs), refs)
	}

	if refs[0].Line != 4 {
		t.Fatalf("expected reference at line 4, got %d", refs[0].Line)
	}

	if filepath.Base(refs[0].File) != "middleware.go" {
		t.Fatalf("expected reference in middleware.go, got %s", refs[0].File)
	}
}

func TestRegexProvider_UnsupportedLanguage(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.rs")
	if err := os.WriteFile(file, []byte("fn main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	prov := &RegexProvider{}
	_, err := prov.Symbols(context.Background(), file)
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func assertSymbol(t *testing.T, m map[string]Symbol, name string, kind SymbolKind, exported bool) {
	t.Helper()
	s, ok := m[name]
	if !ok {
		t.Fatalf("symbol %q not found", name)
	}
	if s.Kind != kind {
		t.Fatalf("symbol %q: kind = %q, want %q", name, s.Kind, kind)
	}
	if s.Exported != exported {
		t.Fatalf("symbol %q: exported = %v, want %v", name, s.Exported, exported)
	}
}
