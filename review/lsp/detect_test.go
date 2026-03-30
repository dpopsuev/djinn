package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectServer(t *testing.T) {
	tests := []struct {
		marker   string
		wantLang string
		wantCmd  string
	}{
		{"go.mod", "go", "gopls"},
		{"Cargo.toml", "rust", "rust-analyzer"},
		{"tsconfig.json", "typescript", "typescript-language-server"},
		{"package.json", "javascript", "typescript-language-server"},
		{"pyproject.toml", "python", "pyright-langserver"},
		{"requirements.txt", "python", "pyright-langserver"},
		{"CMakeLists.txt", "c", "clangd"},
	}
	for _, tt := range tests {
		t.Run(tt.marker, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.marker), []byte(""), 0o644); err != nil {
				t.Fatal(err)
			}

			cfg, ok := DetectServer(dir)
			if !ok {
				t.Fatal("DetectServer returned false, want true")
			}
			if cfg.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", cfg.Language, tt.wantLang)
			}
			if cfg.Command != tt.wantCmd {
				t.Errorf("Command = %q, want %q", cfg.Command, tt.wantCmd)
			}
		})
	}
}

func TestDetectServerNoMatch(t *testing.T) {
	dir := t.TempDir()
	_, ok := DetectServer(dir)
	if ok {
		t.Error("DetectServer on empty dir returned true, want false")
	}
}

func TestDetectServerPriority(t *testing.T) {
	dir := t.TempDir()
	// Create both go.mod and package.json — Go should win (earlier in priority).
	for _, f := range []string{"go.mod", "package.json"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg, ok := DetectServer(dir)
	if !ok {
		t.Fatal("DetectServer returned false, want true")
	}
	if cfg.Language != "go" {
		t.Errorf("Language = %q, want %q (Go should win over JS)", cfg.Language, "go")
	}
	if cfg.Command != "gopls" {
		t.Errorf("Command = %q, want %q", cfg.Command, "gopls")
	}
}

func TestDetectServerReturnsCorrectArgs(t *testing.T) {
	tests := []struct {
		marker   string
		wantArgs []string
	}{
		{"go.mod", []string{"serve"}},
		{"Cargo.toml", nil},
		{"tsconfig.json", []string{"--stdio"}},
		{"CMakeLists.txt", nil},
	}
	for _, tt := range tests {
		t.Run(tt.marker, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.marker), []byte(""), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, ok := DetectServer(dir)
			if !ok {
				t.Fatal("DetectServer returned false")
			}
			if len(cfg.Args) != len(tt.wantArgs) {
				t.Fatalf("Args length = %d, want %d", len(cfg.Args), len(tt.wantArgs))
			}
			for i := range tt.wantArgs {
				if cfg.Args[i] != tt.wantArgs[i] {
					t.Errorf("Args[%d] = %q, want %q", i, cfg.Args[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestDetectServerNonexistentDir(t *testing.T) {
	_, ok := DetectServer("/nonexistent/path/that/does/not/exist")
	if ok {
		t.Error("DetectServer on nonexistent dir returned true, want false")
	}
}
