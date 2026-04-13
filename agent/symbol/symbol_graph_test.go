package symbol

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failProvider always fails — used to test fallback chain.
type failProvider struct{}

func (f *failProvider) Symbols(context.Context, string) ([]Symbol, error) {
	return nil, errors.New("always fails")
}

func (f *failProvider) References(context.Context, string, string) ([]Reference, error) {
	return nil, errors.New("always fails")
}

// staticProvider returns canned results.
type staticProvider struct {
	symbols []Symbol
	refs    map[string][]Reference
}

func (s *staticProvider) Symbols(_ context.Context, _ string) ([]Symbol, error) {
	return s.symbols, nil
}

func (s *staticProvider) References(_ context.Context, _, symbol string) ([]Reference, error) {
	if refs, ok := s.refs[symbol]; ok {
		return refs, nil
	}
	return nil, nil
}

func TestSymbolGraphPopulator_FallbackChain(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // suppress warn/debug in tests
	}))

	static := &staticProvider{
		symbols: []Symbol{
			{Name: "DoWork", Kind: SymbolFunc, Line: 10, Exported: true},
			{Name: "helper", Kind: SymbolFunc, Line: 20, Exported: false},
		},
		refs: map[string][]Reference{
			"DoWork": {
				{File: "/project/cmd/main.go", Line: 42},
				{File: "/project/internal/svc.go", Line: 8},
			},
		},
	}

	// Fail provider first, static second — should fallback.
	pop := NewSymbolGraphPopulator(log, &failProvider{}, static)

	graph, err := pop.Populate(context.Background(), "/project/pkg/handler.go")
	if err != nil {
		t.Fatalf("Populate: %v", err)
	}

	if graph.File != "/project/pkg/handler.go" {
		t.Fatalf("file = %q, want /project/pkg/handler.go", graph.File)
	}

	if len(graph.Symbols) != 2 {
		t.Fatalf("expected 2 symbol entries, got %d", len(graph.Symbols))
	}

	// DoWork should have 2 external callers (different dir from /project/pkg/).
	entry := graph.Symbols[0]
	if entry.Symbol.Name != "DoWork" {
		t.Fatalf("first entry = %q, want DoWork", entry.Symbol.Name)
	}
	totalCallers := len(entry.Internal) + len(entry.External)
	if totalCallers != 2 {
		t.Fatalf("DoWork: expected 2 callers, got %d", totalCallers)
	}

	// helper should have 0 callers.
	entry = graph.Symbols[1]
	if entry.Symbol.Name != "helper" {
		t.Fatalf("second entry = %q, want helper", entry.Symbol.Name)
	}
	if len(entry.Internal)+len(entry.External) != 0 {
		t.Fatal("helper should have 0 callers")
	}
}

func TestSymbolGraph_FormatContext(t *testing.T) {
	graph := &SymbolGraph{
		File: "auth/handler.go",
		Symbols: []SymbolEntry{
			{
				Symbol: Symbol{Name: "ValidateToken", Kind: SymbolFunc, Exported: true},
				Internal: []Reference{
					{File: "auth/middleware.go", Line: 42},
				},
				External: []Reference{
					{File: "cli/repl/model.go", Line: 8},
					{File: "staff/persona.go", Line: 15},
					{File: "broker/dispatch.go", Line: 99},
				},
			},
			{
				Symbol: Symbol{Name: "TokenClaims", Kind: SymbolType, Exported: true},
				External: []Reference{
					{File: "staff/persona.go", Line: 1},
				},
			},
			{
				Symbol: Symbol{Name: "RefreshToken", Kind: SymbolFunc, Exported: true},
			},
		},
	}

	output := graph.FormatContext()

	// Verify structure.
	if !strings.Contains(output, `<symbol-graph file="auth/handler.go">`) {
		t.Fatalf("missing header: %s", output)
	}
	if !strings.Contains(output, "</symbol-graph>") {
		t.Fatalf("missing footer: %s", output)
	}

	// ValidateToken should show 4 callers, first 3 listed.
	if !strings.Contains(output, "ValidateToken (func, 4 callers)") {
		t.Fatalf("missing ValidateToken line: %s", output)
	}
	if !strings.Contains(output, "middleware.go:42") {
		t.Fatalf("missing middleware.go reference: %s", output)
	}
	if !strings.Contains(output, "...") {
		t.Fatalf("missing ellipsis for truncated callers: %s", output)
	}

	// TokenClaims should show 1 caller.
	if !strings.Contains(output, "TokenClaims (type, 1 callers)") {
		t.Fatalf("missing TokenClaims line: %s", output)
	}

	// RefreshToken should show safe to modify.
	if !strings.Contains(output, "RefreshToken (func, 0 callers) | safe to modify") {
		t.Fatalf("missing RefreshToken safe line: %s", output)
	}
}

func TestSymbolGraph_FormatContext_Empty(t *testing.T) {
	// nil graph.
	var sg *SymbolGraph
	if got := sg.FormatContext(); got != "" {
		t.Fatalf("nil graph should return empty string, got %q", got)
	}

	// Empty symbols.
	sg = &SymbolGraph{File: "empty.go"}
	if got := sg.FormatContext(); got != "" {
		t.Fatalf("empty symbols should return empty string, got %q", got)
	}
}

func TestSymbolGraphPopulator_Populate_Integration(t *testing.T) {
	// Integration test: uses real RegexProvider against temp files.
	dir := t.TempDir()

	// Definition file.
	defFile := filepath.Join(dir, "handler.go")
	defSrc := `package handler

func Serve() {}
func helper() {}

type Config struct{}
`
	if err := os.WriteFile(defFile, []byte(defSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Caller file.
	callerFile := filepath.Join(dir, "main.go")
	callerSrc := `package handler

func main() {
	Serve()
	c := Config{}
	_ = c
}
`
	if err := os.WriteFile(callerFile, []byte(callerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	pop := NewSymbolGraphPopulator(log, &RegexProvider{})

	graph, err := pop.Populate(context.Background(), defFile)
	if err != nil {
		t.Fatalf("Populate: %v", err)
	}

	if len(graph.Symbols) < 3 {
		t.Fatalf("expected >= 3 symbols, got %d", len(graph.Symbols))
	}

	// Verify FormatContext produces output.
	output := graph.FormatContext()
	if output == "" {
		t.Fatal("FormatContext should produce output")
	}
	if !strings.Contains(output, "Serve") {
		t.Fatalf("output should contain Serve: %s", output)
	}
}
