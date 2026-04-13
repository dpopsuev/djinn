package symbol

import "testing"

func TestSymbolKind_Constants(t *testing.T) {
	tests := []struct {
		kind SymbolKind
		want string
	}{
		{SymbolFunc, "func"},
		{SymbolType, "type"},
		{SymbolVar, "var"},
		{SymbolConst, "const"},
		{SymbolInterface, "interface"},
		{SymbolMethod, "method"},
	}
	for _, tt := range tests {
		if string(tt.kind) != tt.want {
			t.Fatalf("SymbolKind %q != %q", tt.kind, tt.want)
		}
	}
}

func TestIsExportedGo(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Foo", true},
		{"foo", false},
		{"X", true},
		{"x", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isExportedGo(tt.name); got != tt.want {
			t.Fatalf("isExportedGo(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
