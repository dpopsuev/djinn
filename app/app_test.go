package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestRun_Routing(t *testing.T) {
	tests := []struct {
		args    []string
		want    string
		wantErr bool
	}{
		{[]string{"version"}, "djinn " + Version, false},
		{[]string{"--help"}, "djinn", false},
		{[]string{"-h"}, "djinn", false},
		{[]string{"help"}, "djinn", false},
	}
	for _, tt := range tests {
		var buf bytes.Buffer
		err := Run(tt.args, &buf)
		if tt.wantErr && err == nil {
			t.Fatalf("args=%v: expected error", tt.args)
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("args=%v: %v", tt.args, err)
		}
		if !strings.Contains(buf.String(), tt.want) {
			t.Fatalf("args=%v: output=%q, want %q", tt.args, buf.String(), tt.want)
		}
	}
}

func TestRunKill_MissingArgs(t *testing.T) {
	var buf bytes.Buffer
	err := RunKill(nil, &buf)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestRunAttach_MissingArgs(t *testing.T) {
	var buf bytes.Buffer
	err := RunAttach(nil, &buf)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestRunImport_MissingArgs(t *testing.T) {
	var buf bytes.Buffer
	err := RunImport(nil, &buf)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestRunDoctor_Output(t *testing.T) {
	var buf bytes.Buffer
	err := RunDoctor(&buf)
	if err != nil {
		t.Fatalf("RunDoctor: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "djinn doctor") {
		t.Fatal("missing header")
	}
	if !strings.Contains(out, "version:") {
		t.Fatal("missing version")
	}
	if !strings.Contains(out, "drivers:") {
		t.Fatal("missing drivers section")
	}
}

func TestPrintUsage_ContainsMode(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	if !strings.Contains(buf.String(), "--mode") {
		t.Fatal("usage should mention --mode flag")
	}
}

func TestPrintUsage_ContainsConfig(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	if !strings.Contains(buf.String(), "--config") {
		t.Fatal("usage should mention --config flag")
	}
	if !strings.Contains(buf.String(), "config dump") {
		t.Fatal("usage should mention config dump subcommand")
	}
}

func TestRunConfig_Dump(t *testing.T) {
	var buf bytes.Buffer
	err := RunConfig([]string{"dump"}, &buf)
	if err != nil {
		t.Fatalf("RunConfig dump: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "mode:") {
		t.Fatal("dump should contain mode")
	}
	if !strings.Contains(out, "driver:") {
		t.Fatal("dump should contain driver")
	}
}

func TestRunConfig_MissingArgs(t *testing.T) {
	var buf bytes.Buffer
	err := RunConfig(nil, &buf)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestRunConfig_Unknown(t *testing.T) {
	var buf bytes.Buffer
	err := RunConfig([]string{"bogus"}, &buf)
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestDefaultHomeDir(t *testing.T) {
	if DefaultHomeDir != ".djinn" {
		t.Fatalf("DefaultHomeDir = %q, want .djinn", DefaultHomeDir)
	}
	if DefaultSessionDir != ".djinn/sessions" {
		t.Fatalf("DefaultSessionDir = %q, want .djinn/sessions", DefaultSessionDir)
	}
}
