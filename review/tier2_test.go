package review

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/review/lsp"
)

// mockDiffer implements SymbolDiffer for testing.
type mockDiffer struct {
	diffs map[string]*SymbolDiff // file path → diff
}

func (m *mockDiffer) DiffFile(_ context.Context, _, filePath string) (*SymbolDiff, error) {
	if d, ok := m.diffs[filePath]; ok {
		return d, nil
	}
	return nil, nil // unsupported file
}

func TestDiffSymbolSets_AddedRemovedModified(t *testing.T) {
	before := []SymbolInfo{
		{Name: "Foo", Kind: lsp.SymbolFunction, Exported: true, Detail: "(int) error"},
		{Name: "Bar", Kind: lsp.SymbolStruct, Exported: true, Children: 3},
		{Name: "Old", Kind: lsp.SymbolFunction, Exported: true},
	}
	after := []SymbolInfo{
		{Name: "Foo", Kind: lsp.SymbolFunction, Exported: true, Detail: "(string) error"}, // modified
		{Name: "Bar", Kind: lsp.SymbolStruct, Exported: true, Children: 5},                // modified (fields)
		{Name: "New", Kind: lsp.SymbolFunction, Exported: true},                           // added
	}

	diff := DiffSymbolSets(before, after)

	if len(diff.Added) != 1 || diff.Added[0].Name != "New" {
		t.Errorf("Added = %v, want [New]", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Name != "Old" {
		t.Errorf("Removed = %v, want [Old]", diff.Removed)
	}
	if len(diff.Modified) != 2 {
		t.Fatalf("Modified = %d, want 2", len(diff.Modified))
	}
}

func TestDiffSymbolSets_NoChanges(t *testing.T) {
	symbols := []SymbolInfo{
		{Name: "Foo", Kind: lsp.SymbolFunction, Detail: "(int) error"},
	}
	diff := DiffSymbolSets(symbols, symbols)

	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Modified) != 0 {
		t.Error("identical sets should produce empty diff")
	}
}

func TestSymbolsFromLSP(t *testing.T) {
	lspSyms := []lsp.DocumentSymbol{
		{Name: "CreateOrder", Kind: lsp.SymbolFunction, Detail: "(req OrderRequest) (Order, error)", Children: nil},
		{Name: "Repository", Kind: lsp.SymbolInterface, Children: []lsp.DocumentSymbol{{Name: "Save"}, {Name: "Find"}, {Name: "Delete"}}},
		{Name: "config", Kind: lsp.SymbolStruct, Children: []lsp.DocumentSymbol{{Name: "Host"}, {Name: "Port"}}},
	}

	infos := SymbolsFromLSP(lspSyms)
	if len(infos) != 3 {
		t.Fatalf("got %d symbols, want 3", len(infos))
	}
	if !infos[0].Exported {
		t.Error("CreateOrder should be exported (uppercase)")
	}
	if infos[1].Children != 3 {
		t.Errorf("Repository children = %d, want 3", infos[1].Children)
	}
	if infos[2].Exported {
		t.Error("config should not be exported (lowercase)")
	}
}

func TestExportsHeuristic_Exceeded(t *testing.T) {
	differ := &mockDiffer{
		diffs: map[string]*SymbolDiff{
			"handler.go": {
				Added:    []SymbolInfo{{Name: "NewFunc", Exported: true}, {Name: "AnotherFunc", Exported: true}},
				Removed:  []SymbolInfo{{Name: "OldFunc", Exported: true}},
				Modified: []SymbolChange{{Name: "UpdateFunc", Exported: true}},
			},
			"store.go": {
				Added: []SymbolInfo{{Name: "Save", Exported: true}, {Name: "Delete", Exported: true}},
			},
		},
	}
	cfg := DefaultConfig()
	cfg.MaxExportedChanges = 5
	h := NewExportsHeuristic(cfg, differ)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"handler.go", "store.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// 2 added + 1 removed + 1 modified + 2 added = 6 >= 5
	if !signals[0].Exceeded {
		t.Errorf("expected exceeded (6 >= 5), value=%.0f", signals[0].Value)
	}
}

func TestExportsHeuristic_SkipsUnexported(t *testing.T) {
	differ := &mockDiffer{
		diffs: map[string]*SymbolDiff{
			"handler.go": {
				Added: []SymbolInfo{
					{Name: "privateFunc", Exported: false},
					{Name: "PublicFunc", Exported: true},
				},
			},
		},
	}
	cfg := DefaultConfig()
	cfg.MaxExportedChanges = 5
	h := NewExportsHeuristic(cfg, differ)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"handler.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only 1 exported change (PublicFunc), not exceeded
	if signals[0].Exceeded {
		t.Error("1 exported change should not exceed threshold 5")
	}
}

func TestInterfaceHeuristic_NewInterface(t *testing.T) {
	differ := &mockDiffer{
		diffs: map[string]*SymbolDiff{
			"port.go": {
				Added: []SymbolInfo{
					{Name: "Repository", Kind: lsp.SymbolInterface, Exported: true, Children: 3},
					{Name: "Cache", Kind: lsp.SymbolInterface, Exported: true, Children: 2},
				},
			},
		},
	}
	cfg := DefaultConfig()
	cfg.MaxInterfacesIntroduced = 2
	h := NewInterfaceHeuristic(cfg, differ)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"port.go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	exceeded := Exceeded(signals)
	if len(exceeded) != 1 {
		t.Fatalf("expected 1 exceeded signal, got %d", len(exceeded))
	}
	if exceeded[0].Metric != "interfaces_introduced" {
		t.Errorf("metric = %q, want interfaces_introduced", exceeded[0].Metric)
	}
}

func TestInterfaceHeuristic_MethodChange(t *testing.T) {
	differ := &mockDiffer{
		diffs: map[string]*SymbolDiff{
			"port.go": {
				Modified: []SymbolChange{
					{Name: "Repository", Kind: lsp.SymbolInterface, ChildrenBefore: 3, ChildrenAfter: 4},
				},
			},
		},
	}
	cfg := DefaultConfig()
	cfg.OnInterfaceChange = true
	h := NewInterfaceHeuristic(cfg, differ)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"port.go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Interface method change = boolean trigger
	var methodSig *Signal
	for i := range signals {
		if signals[i].Metric == "interface_methods_changed" {
			methodSig = &signals[i]
			break
		}
	}
	if methodSig == nil {
		t.Fatal("missing interface_methods_changed signal")
	}
	if !methodSig.Exceeded {
		t.Error("interface method change should always trigger")
	}
}

func TestSignatureHeuristic_Exceeded(t *testing.T) {
	differ := &mockDiffer{
		diffs: map[string]*SymbolDiff{
			"handler.go": {
				Modified: []SymbolChange{
					{Name: "Create", Kind: lsp.SymbolFunction, DetailBefore: "(int) error", DetailAfter: "(string) error"},
					{Name: "Update", Kind: lsp.SymbolFunction, DetailBefore: "(int)", DetailAfter: "(int, bool)"},
					{Name: "Delete", Kind: lsp.SymbolMethod, DetailBefore: "(id int)", DetailAfter: "(id string)"},
				},
			},
			"store.go": {
				Modified: []SymbolChange{
					{Name: "Save", Kind: lsp.SymbolFunction, DetailBefore: "(x)", DetailAfter: "(x, y)"},
					{Name: "Load", Kind: lsp.SymbolFunction, DetailBefore: "()", DetailAfter: "(ctx)"},
				},
			},
		},
	}
	cfg := DefaultConfig()
	cfg.MaxSignatureChanges = 5
	h := NewSignatureHeuristic(cfg, differ)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"handler.go", "store.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// 5 signature changes = 5 >= 5 threshold
	if !signals[0].Exceeded {
		t.Errorf("5 changes should exceed threshold 5, value=%.0f", signals[0].Value)
	}
}

func TestSignatureHeuristic_IgnoresNonFunctions(t *testing.T) {
	differ := &mockDiffer{
		diffs: map[string]*SymbolDiff{
			"types.go": {
				Modified: []SymbolChange{
					{Name: "Config", Kind: lsp.SymbolStruct, DetailBefore: "struct", DetailAfter: "struct"},
				},
			},
		},
	}
	cfg := DefaultConfig()
	cfg.MaxSignatureChanges = 1
	h := NewSignatureHeuristic(cfg, differ)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"types.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if signals[0].Exceeded {
		t.Error("struct changes should not trigger signature heuristic")
	}
}

func TestDetectServer(t *testing.T) {
	// Test that DetectServer returns false for empty dir.
	cfg, found := lsp.DetectServer(t.TempDir())
	if found {
		t.Errorf("should not find server in empty dir, got %+v", cfg)
	}
}

func TestExportsHeuristic_NilDiffer(t *testing.T) {
	cfg := DefaultConfig()
	h := NewExportsHeuristic(cfg, nil)

	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"handler.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 0 {
		t.Error("nil differ should produce 0 signals")
	}
}
