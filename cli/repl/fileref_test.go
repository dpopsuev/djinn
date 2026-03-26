package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// RED: Error cases
// ═══════════════════════════════════════════════════════════════════════

func TestPreprocessFileRefs_MissingFile(t *testing.T) {
	result := preprocessFileRefs("look at @nonexistent.go", t.TempDir())
	if !strings.Contains(result, "error=") {
		t.Fatalf("missing file should produce error tag: %q", result)
	}
	if !strings.Contains(result, "nonexistent.go") {
		t.Fatal("should preserve file path in error")
	}
}

func TestPreprocessFileRefs_NoRefs(t *testing.T) {
	prompt := "just a normal prompt with no references"
	result := preprocessFileRefs(prompt, t.TempDir())
	if result != prompt {
		t.Fatalf("prompt should be unchanged: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GREEN: Happy path
// ═══════════════════════════════════════════════════════════════════════

func TestPreprocessFileRefs_SingleFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)

	result := preprocessFileRefs("review @main.go", dir)
	if !strings.Contains(result, "<file path=") {
		t.Fatalf("missing file tag: %q", result)
	}
	if !strings.Contains(result, "package main") {
		t.Fatalf("missing file content: %q", result)
	}
	if !strings.Contains(result, "</file>") {
		t.Fatal("missing closing tag")
	}
}

func TestPreprocessFileRefs_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("file a\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("file b\n"), 0644)

	result := preprocessFileRefs("compare @a.go and @b.go", dir)
	if strings.Count(result, "<file path=") != 2 {
		t.Fatalf("expected 2 file tags, got %d in: %q", strings.Count(result, "<file path="), result)
	}
	if !strings.Contains(result, "file a") || !strings.Contains(result, "file b") {
		t.Fatal("missing content from one or both files")
	}
}

func TestPreprocessFileRefs_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "abs.txt")
	os.WriteFile(absPath, []byte("absolute content\n"), 0644)

	result := preprocessFileRefs("read @"+absPath, "/some/other/dir")
	if !strings.Contains(result, "absolute content") {
		t.Fatalf("absolute path should resolve: %q", result)
	}
}

func TestPreprocessFileRefs_RelativePath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "rel.txt"), []byte("relative content\n"), 0644)

	result := preprocessFileRefs("check @rel.txt", dir)
	if !strings.Contains(result, "relative content") {
		t.Fatalf("relative path should resolve against workDir: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BLUE: Edge cases
// ═══════════════════════════════════════════════════════════════════════

func TestPreprocessFileRefs_AtInEmail(t *testing.T) {
	// user@email.com should NOT be treated as a file reference.
	prompt := "contact user@email.com for details"
	result := preprocessFileRefs(prompt, t.TempDir())
	// The @ is mid-word, so "user@email.com" is one token starting with "user" not "@".
	if result != prompt {
		t.Fatalf("email should not trigger file ref: %q", result)
	}
}

func TestPreprocessFileRefs_AtAlone(t *testing.T) {
	prompt := "just @ nothing"
	result := preprocessFileRefs(prompt, t.TempDir())
	// Bare "@" has len("") == 0 after trim, should be skipped.
	if strings.Contains(result, "<file") {
		t.Fatalf("bare @ should not trigger: %q", result)
	}
}

func TestPreprocessFileRefs_MixedContent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("main\n"), 0644)
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("test\n"), 0644)

	result := preprocessFileRefs("review @main.go and also check @test.go please", dir)
	if strings.Count(result, "<file path=") != 2 {
		t.Fatalf("expected 2 file refs: %q", result)
	}
	// Original prompt should be preserved.
	if !strings.HasPrefix(result, "review @main.go") {
		t.Fatal("original prompt should be prefix")
	}
}

func TestPreprocessFileRefs_TrailingPunctuation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.go"), []byte("content\n"), 0644)

	// @file.go, or @file.go. should still resolve.
	result := preprocessFileRefs("check @file.go, please", dir)
	if !strings.Contains(result, "content") {
		t.Fatalf("trailing comma should be stripped: %q", result)
	}
}
