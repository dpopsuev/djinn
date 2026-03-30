package lsp

import "testing"

func TestSymbolKindString(t *testing.T) {
	tests := []struct {
		kind SymbolKind
		want string
	}{
		{SymbolFile, "File"},
		{SymbolModule, "Module"},
		{SymbolNamespace, "Namespace"},
		{SymbolPackage, "Package"},
		{SymbolClass, "Class"},
		{SymbolMethod, "Method"},
		{SymbolProperty, "Property"},
		{SymbolField, "Field"},
		{SymbolConstructor, "Constructor"},
		{SymbolEnum, "Enum"},
		{SymbolInterface, "Interface"},
		{SymbolFunction, "Function"},
		{SymbolVariable, "Variable"},
		{SymbolConstant, "Constant"},
		{SymbolString, "String"},
		{SymbolStruct, "Struct"},
		{SymbolTypeParameter, "TypeParameter"},
		// These are valid LSP kinds but not in the names map — expect "Unknown".
		{SymbolNumber, "Unknown"},
		{SymbolBoolean, "Unknown"},
		{SymbolArray, "Unknown"},
		{SymbolObject, "Unknown"},
		{SymbolKey, "Unknown"},
		{SymbolNull, "Unknown"},
		{SymbolEnumMember, "Unknown"},
		{SymbolEvent, "Unknown"},
		{SymbolOperator, "Unknown"},
		// Out-of-range sentinel.
		{SymbolKind(999), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.kind.String()
			if got != tt.want {
				t.Errorf("SymbolKind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
			}
		})
	}
}

func TestSymbolKindIsExportable(t *testing.T) {
	exportable := map[SymbolKind]bool{
		SymbolFile:          false,
		SymbolModule:        false,
		SymbolNamespace:     false,
		SymbolPackage:       false,
		SymbolClass:         true,
		SymbolMethod:        true,
		SymbolProperty:      true,
		SymbolField:         true,
		SymbolConstructor:   false,
		SymbolEnum:          true,
		SymbolInterface:     true,
		SymbolFunction:      true,
		SymbolVariable:      true,
		SymbolConstant:      true,
		SymbolString:        false,
		SymbolNumber:        false,
		SymbolBoolean:       false,
		SymbolArray:         false,
		SymbolObject:        false,
		SymbolKey:           false,
		SymbolNull:          false,
		SymbolEnumMember:    false,
		SymbolStruct:        true,
		SymbolEvent:         false,
		SymbolOperator:      false,
		SymbolTypeParameter: true,
	}
	for kind, want := range exportable {
		got := kind.IsExportable()
		if got != want {
			t.Errorf("SymbolKind(%d).IsExportable() = %v, want %v", int(kind), got, want)
		}
	}

	// Sentinel: unknown kind should not be exportable.
	if SymbolKind(999).IsExportable() {
		t.Error("SymbolKind(999).IsExportable() = true, want false")
	}
}

func TestFlattenSymbols(t *testing.T) {
	// Build a hierarchy:
	//   ClassA
	//     MethodB
	//       FieldC
	//   FunctionD
	symbols := []DocumentSymbol{
		{
			Name: "ClassA",
			Kind: SymbolClass,
			Children: []DocumentSymbol{
				{
					Name: "MethodB",
					Kind: SymbolMethod,
					Children: []DocumentSymbol{
						{Name: "FieldC", Kind: SymbolField},
					},
				},
			},
		},
		{
			Name: "FunctionD",
			Kind: SymbolFunction,
		},
	}

	flat := FlattenSymbols(symbols)
	wantNames := []string{"ClassA", "MethodB", "FieldC", "FunctionD"}

	if len(flat) != len(wantNames) {
		t.Fatalf("FlattenSymbols returned %d symbols, want %d", len(flat), len(wantNames))
	}
	for i, want := range wantNames {
		if flat[i].Name != want {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, want)
		}
	}
}

func TestFlattenSymbolsEmpty(t *testing.T) {
	flat := FlattenSymbols(nil)
	if len(flat) != 0 {
		t.Errorf("FlattenSymbols(nil) returned %d symbols, want 0", len(flat))
	}

	flat = FlattenSymbols([]DocumentSymbol{})
	if len(flat) != 0 {
		t.Errorf("FlattenSymbols([]) returned %d symbols, want 0", len(flat))
	}
}

func TestFlattenSymbolsPreservesFields(t *testing.T) {
	sym := DocumentSymbol{
		Name:   "Foo",
		Detail: "func Foo()",
		Kind:   SymbolFunction,
		Range: Range{
			Start: Position{Line: 10, Character: 0},
			End:   Position{Line: 20, Character: 1},
		},
		SelectionRange: Range{
			Start: Position{Line: 10, Character: 5},
			End:   Position{Line: 10, Character: 8},
		},
	}

	flat := FlattenSymbols([]DocumentSymbol{sym})
	if len(flat) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(flat))
	}
	got := flat[0]
	if got.Detail != "func Foo()" {
		t.Errorf("Detail = %q, want %q", got.Detail, "func Foo()")
	}
	if got.Range.Start.Line != 10 || got.Range.End.Line != 20 {
		t.Errorf("Range not preserved: got %+v", got.Range)
	}
}
