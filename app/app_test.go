package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/session"
)

func TestSessionDir(t *testing.T) {
	dir := SessionDir()
	if dir == "" {
		t.Fatal("SessionDir returned empty")
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("SessionDir not absolute: %q", dir)
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		t.Skip("no HOME set")
	}
	expected := filepath.Join(home, DefaultSessionDir)
	if dir != expected {
		t.Fatalf("SessionDir = %q, want %q", dir, expected)
	}
}

func TestGetwd(t *testing.T) {
	d := Getwd()
	if d == "" {
		t.Fatal("Getwd returned empty")
	}
}

func TestCreateDriver_Claude(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	d, err := CreateDriver(DriverClaude, "claude-sonnet-4-6", "")
	if err != nil {
		t.Fatalf("CreateDriver claude: %v", err)
	}
	if d == nil {
		t.Fatal("driver is nil")
	}
}

func TestCreateDriver_UnknownDriver(t *testing.T) {
	_, err := CreateDriver("unknown", "model", "")
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}

func TestCreateDriver_OllamaNotImplemented(t *testing.T) {
	_, err := CreateDriver(DriverOllama, "qwen2.5", "")
	if err == nil {
		t.Fatal("expected error for ollama (not yet implemented)")
	}
}

func TestLoadMostRecent_Empty(t *testing.T) {
	store, _ := session.NewStore(t.TempDir())
	_, err := LoadMostRecent(store)
	if err == nil {
		t.Fatal("expected error for empty session store")
	}
}

func TestLoadMostRecent_WithSessions(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	s1 := session.New("s1", "model", "/work")
	s1.Name = "older"
	s1.Append(session.Entry{Content: "old"})
	store.Save(s1)

	s2 := session.New("s2", "model", "/work")
	s2.Name = "newer"
	s2.Append(session.Entry{Content: "new"})
	store.Save(s2)

	loaded, err := LoadMostRecent(store)
	if err != nil {
		t.Fatalf("LoadMostRecent: %v", err)
	}
	if loaded.Name != "newer" {
		t.Fatalf("loaded = %q, want %q (most recent)", loaded.Name, "newer")
	}
}

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	if buf.Len() == 0 {
		t.Fatal("PrintUsage should produce output")
	}
}

func TestReadSystemFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "system.txt")
	os.WriteFile(path, []byte("You are a Go expert."), 0644)

	content := ReadSystemFile(path)
	if content != "You are a Go expert." {
		t.Fatalf("content = %q", content)
	}
}

func TestReadSystemFile_Missing(t *testing.T) {
	content := ReadSystemFile("/nonexistent/file.txt")
	if content != "" {
		t.Fatalf("missing file should return empty, got %q", content)
	}
}

func TestReadSystemFile_Empty(t *testing.T) {
	content := ReadSystemFile("")
	if content != "" {
		t.Fatal("empty path should return empty")
	}
}

func TestRun_Version(t *testing.T) {
	var buf bytes.Buffer
	if err := Run([]string{"version"}, &buf); err != nil {
		t.Fatalf("version: %v", err)
	}
	if got := buf.String(); got != "djinn "+Version+"\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestRun_Help(t *testing.T) {
	var buf bytes.Buffer
	if err := Run([]string{"--help"}, &buf); err != nil {
		t.Fatalf("help: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("help output should not be empty")
	}
}
