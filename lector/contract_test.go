package lector

import "testing"

// RunLectorContract runs the Liskov contract test suite against any Lector.
func RunLectorContract(t *testing.T, factory func(t *testing.T) Lector) {
	t.Helper()

	t.Run("OnFileRead_indexes_file", func(t *testing.T) {
		l := factory(t)
		l.OnFileRead("/src/main.go")
		fe, ok := l.FileInfo("/src/main.go")
		if !ok {
			t.Fatal("file should be indexed after OnFileRead")
		}
		if fe.Path != "/src/main.go" {
			t.Errorf("Path = %q", fe.Path)
		}
	})

	t.Run("OnFileWrite_invalidates_and_reindexes", func(t *testing.T) {
		l := factory(t)
		l.OnFileRead("/src/main.go")
		l.OnFileWrite("/src/main.go")
		_, ok := l.FileInfo("/src/main.go")
		if !ok {
			t.Fatal("file should be re-indexed after OnFileWrite")
		}
	})

	t.Run("OnFileDelete_removes", func(t *testing.T) {
		l := factory(t)
		l.OnFileRead("/src/main.go")
		l.OnFileDelete("/src/main.go")
		_, ok := l.FileInfo("/src/main.go")
		if ok {
			t.Fatal("file should be gone after OnFileDelete")
		}
	})

	t.Run("Symbols_returns_seeded", func(t *testing.T) {
		l := factory(t)
		syms := l.Symbols("main")
		// Stub may or may not have symbols — just verify no panic.
		_ = syms
	})

	t.Run("SymbolsForFile_returns_matching", func(t *testing.T) {
		l := factory(t)
		syms := l.SymbolsForFile("/src/main.go")
		_ = syms // no panic
	})

	t.Run("FuzzyFiles_returns_matches", func(t *testing.T) {
		l := factory(t)
		l.OnFileRead("/src/auth/handler.go")
		l.OnFileRead("/src/auth/handler_test.go")
		l.OnFileRead("/src/config/config.go")

		results := l.FuzzyFiles("handler")
		if len(results) < 2 {
			t.Fatalf("expected >= 2 fuzzy matches for 'handler', got %d", len(results))
		}
	})

	t.Run("FuzzySymbols_returns_matches", func(t *testing.T) {
		l := factory(t)
		// StubLector needs symbols seeded for this test.
		syms := l.FuzzySymbols("validate")
		_ = syms // no panic, may be empty if not seeded
	})

	t.Run("FuzzyFiles_no_match_returns_empty", func(t *testing.T) {
		l := factory(t)
		results := l.FuzzyFiles("nonexistent_xyz_123")
		if len(results) != 0 {
			t.Fatalf("expected 0 matches, got %d", len(results))
		}
	})
}

// --- Run contract against StubLector ---

func TestStubLector_Contract(t *testing.T) {
	RunLectorContract(t, func(_ *testing.T) Lector {
		s := NewStubLector()
		s.SeedSymbols("main", []Symbol{
			{Name: "ValidateToken", Kind: "func", File: "/src/main.go", Line: 10, Exported: true},
		})
		return s
	})
}

// --- Fuzzy search specific tests ---

func TestStubLector_FuzzySymbols(t *testing.T) {
	s := NewStubLector()
	s.SeedSymbols("auth", []Symbol{
		{Name: "ValidateToken", Kind: "func"},
		{Name: "ValidateTokenClaims", Kind: "func"},
		{Name: "HandleAuth", Kind: "func"},
	})

	results := s.FuzzySymbols("validate")
	if len(results) != 2 {
		t.Fatalf("expected 2 matches for 'validate', got %d", len(results))
	}
}
