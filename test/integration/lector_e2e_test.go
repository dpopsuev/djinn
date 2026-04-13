//go:build integration

package integration

import (
	"testing"

	"github.com/dpopsuev/djinn/lector"
	"github.com/dpopsuev/djinn/substrate"
)

// TestLector_E2E_ObserveAndQuery proves the full Lector pipeline:
// OnFileRead → file indexed → SymbolsForFile cached → FuzzyFiles works.
func TestLector_E2E_ObserveAndQuery(t *testing.T) {
	dir := t.TempDir()

	// Seed Lector with symbols (StubLector auto-created by Substrate).
	stub := lector.NewStubLector()
	stub.SeedSymbols("main", []lector.Symbol{
		{Name: "ValidateToken", Kind: "func", File: "/src/main.go", Line: 10, Exported: true},
		{Name: "HandleAuth", Kind: "func", File: "/src/auth.go", Line: 5, Exported: true},
		{Name: "newRouter", Kind: "func", File: "/src/main.go", Line: 20, Exported: false},
	})

	sub := substrate.New(dir, substrate.WithLector(stub))

	l := sub.Lector()
	if l == nil {
		t.Fatal("Lector should not be nil")
	}

	// Observe a file read.
	l.OnFileRead("/src/main.go")
	l.OnFileRead("/src/auth.go")
	l.OnFileRead("/src/config.yaml")

	// File index queries.
	fe, ok := l.FileInfo("/src/main.go")
	if !ok {
		t.Fatal("main.go should be indexed")
	}
	if fe.Path != "/src/main.go" {
		t.Errorf("Path = %q", fe.Path)
	}

	// Fuzzy file search.
	files := l.FuzzyFiles("auth")
	if len(files) != 1 {
		t.Fatalf("FuzzyFiles('auth') = %d, want 1", len(files))
	}

	// Fuzzy symbol search.
	syms := l.FuzzySymbols("validate")
	if len(syms) != 1 {
		t.Fatalf("FuzzySymbols('validate') = %d, want 1", len(syms))
	}
	if syms[0].Name != "ValidateToken" {
		t.Errorf("symbol = %q, want ValidateToken", syms[0].Name)
	}

	// SymbolsForFile.
	mainSyms := l.SymbolsForFile("/src/main.go")
	if len(mainSyms) != 2 { // ValidateToken + newRouter
		t.Fatalf("SymbolsForFile(main.go) = %d, want 2", len(mainSyms))
	}

	// Delete removes from index.
	l.OnFileDelete("/src/config.yaml")
	_, ok = l.FileInfo("/src/config.yaml")
	if ok {
		t.Fatal("config.yaml should be gone after delete")
	}
}

// TestLector_E2E_SubstrateDefaultsToStub proves Substrate auto-creates StubLector.
func TestLector_E2E_SubstrateDefaultsToStub(t *testing.T) {
	sub := substrate.New(t.TempDir())

	l := sub.Lector()
	if l == nil {
		t.Fatal("Lector should be auto-created")
	}

	// Should work without panic.
	l.OnFileRead("/tmp/test.go")
	_, ok := l.FileInfo("/tmp/test.go")
	if !ok {
		t.Fatal("default StubLector should index files")
	}
}
