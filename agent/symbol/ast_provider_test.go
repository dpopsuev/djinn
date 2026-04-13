package symbol

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestASTProvider_Symbols_GoFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	src := `package sample

import "fmt"

func PublicFunc() {
	fmt.Println("hello")
}

func privateFunc() {}

type MyStruct struct {
	Field int
}

func (m *MyStruct) Method() string {
	return "hello"
}

type MyInterface interface {
	Do()
}

var GlobalVar = 42

const MaxRetries = 3

const (
	A = 1
	B = 2
)
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	prov := NewASTProvider()
	symbols, err := prov.Symbols(context.Background(), file)
	if err != nil {
		t.Fatalf("Symbols: %v", err)
	}

	byName := make(map[string]Symbol)
	for _, s := range symbols {
		byName[s.Name] = s
	}

	// Check declarations.
	assertSymbol(t, byName, "PublicFunc", SymbolFunc, true)
	assertSymbol(t, byName, "privateFunc", SymbolFunc, false)
	assertSymbol(t, byName, "MyStruct", SymbolType, true)
	assertSymbol(t, byName, "Method", SymbolMethod, true)
	assertSymbol(t, byName, "MyInterface", SymbolInterface, true)
	assertSymbol(t, byName, "GlobalVar", SymbolVar, true)
	assertSymbol(t, byName, "MaxRetries", SymbolConst, true)
	assertSymbol(t, byName, "A", SymbolConst, true)
	assertSymbol(t, byName, "B", SymbolConst, true)

	// Verify method is correctly classified.
	if byName["Method"].Kind != SymbolMethod {
		t.Fatalf("Method should be SymbolMethod, got %s", byName["Method"].Kind)
	}
}

func TestASTProvider_NonGoFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.py")
	if err := os.WriteFile(file, []byte("def foo(): pass"), 0o644); err != nil {
		t.Fatal(err)
	}

	prov := NewASTProvider()
	_, err := prov.Symbols(context.Background(), file)
	if err == nil {
		t.Fatal("expected error for non-Go file")
	}
}

func TestASTProvider_BlankIdentifier_Skipped(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "blank.go")
	src := `package blank

var _ = "unused"
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	prov := NewASTProvider()
	symbols, err := prov.Symbols(context.Background(), file)
	if err != nil {
		t.Fatalf("Symbols: %v", err)
	}

	for _, s := range symbols {
		if s.Name == "_" {
			t.Fatal("blank identifier should be skipped")
		}
	}
}
