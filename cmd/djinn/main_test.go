package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/session"
)

func TestSessionDir(t *testing.T) {
	dir := sessionDir()
	if dir == "" {
		t.Fatal("sessionDir returned empty")
	}
	home, _ := os.UserHomeDir()
	if !filepath.IsAbs(dir) {
		t.Fatalf("sessionDir not absolute: %q", dir)
	}
	if home == "" {
		t.Skip("no HOME set")
	}
	expected := filepath.Join(home, defaultSessionDir)
	if dir != expected {
		t.Fatalf("sessionDir = %q, want %q", dir, expected)
	}
}

func TestMustGetwd(t *testing.T) {
	d := mustGetwd()
	if d == "" {
		t.Fatal("mustGetwd returned empty")
	}
}

func TestCreateDriver_Claude(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	d, err := createDriver(driverClaude, "claude-sonnet-4-6", "")
	if err != nil {
		t.Fatalf("createDriver claude: %v", err)
	}
	if d == nil {
		t.Fatal("driver is nil")
	}
}

func TestCreateDriver_UnknownDriver(t *testing.T) {
	_, err := createDriver("unknown", "model", "")
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}

func TestCreateDriver_OllamaNotImplemented(t *testing.T) {
	_, err := createDriver(driverOllama, "qwen2.5", "")
	if err == nil {
		t.Fatal("expected error for ollama (not yet implemented)")
	}
}

func TestLoadMostRecent_Empty(t *testing.T) {
	store, _ := session.NewStore(t.TempDir())
	_, err := loadMostRecent(store)
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

	loaded, err := loadMostRecent(store)
	if err != nil {
		t.Fatalf("loadMostRecent: %v", err)
	}
	if loaded.Name != "newer" {
		t.Fatalf("loaded = %q, want %q (most recent)", loaded.Name, "newer")
	}
}

func TestPrintUsage_NoCrash(t *testing.T) {
	// Verify printUsage doesn't panic
	// Redirecting stderr to avoid noise in test output
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = old }()
	printUsage()
}
