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
	if !strings.HasSuffix(dir, "sessions") {
		t.Fatalf("SessionDir should end with /sessions: %q", dir)
	}
}

func TestHomeDir_DualPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Neither exists → default to XDG ~/.config/djinn
	dir := HomeDir()
	if !strings.Contains(dir, ".config/djinn") {
		t.Fatalf("default should be .config/djinn (XDG), got %q", dir)
	}

	// Create ~/.djinn (legacy) → should find it
	djinnDir := filepath.Join(home, ".djinn")
	os.MkdirAll(djinnDir, 0755)
	dir = HomeDir()
	if dir != djinnDir {
		t.Fatalf("should find legacy .djinn, got %q", dir)
	}

	// Create ~/.config/djinn (XDG) → should prefer it
	configDir := filepath.Join(home, ".config", "djinn")
	os.MkdirAll(configDir, 0755)
	dir = HomeDir()
	if dir != configDir {
		t.Fatalf("should prefer XDG .config/djinn, got %q", dir)
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

func TestRunAttach_NoArgs_ShowsTelescope(t *testing.T) {
	var buf bytes.Buffer
	err := RunAttach(nil, &buf)
	if err != nil {
		t.Fatalf("telescope should not error: %v", err)
	}
	// Should show "no sessions" or session list
	out := buf.String()
	if out == "" {
		t.Fatal("should produce output")
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

func TestCreateDriver_Cursor(t *testing.T) {
	d, err := CreateDriver(DriverCursor, "sonnet-4", "")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("cursor driver nil")
	}
}

func TestCreateDriver_Gemini(t *testing.T) {
	d, err := CreateDriver("gemini", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("gemini driver nil")
	}
}

func TestCreateDriver_Codex(t *testing.T) {
	d, err := CreateDriver("codex", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("codex driver nil")
	}
}

func TestCreateDriver_ACP(t *testing.T) {
	d, err := CreateDriver("acp", "cursor/sonnet-4", "")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("acp driver nil")
	}
}

func TestCreateDriver_ACP_ModelSplit(t *testing.T) {
	d, err := CreateDriver("acp", "gemini/gemini-2", "")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("acp driver nil")
	}
}

func TestCreateDriver_WithLogger(t *testing.T) {
	d, err := CreateDriver("cursor", "sonnet-4", "be helpful", nil)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("driver nil")
	}
}

func TestDefaultHubSocket(t *testing.T) {
	path := DefaultHubSocket()
	if !strings.Contains(path, "hub.sock") {
		t.Fatalf("path = %q, want hub.sock", path)
	}
}

func TestRun_Debug_NoArgs(t *testing.T) {
	var buf bytes.Buffer
	Run([]string{"debug"}, &buf)
	if !strings.Contains(buf.String(), "debug") {
		t.Fatal("debug no args should show help")
	}
}

func TestRun_Ls(t *testing.T) {
	var buf bytes.Buffer
	Run([]string{"ls"}, &buf) //nolint:errcheck
}

func TestRunDebug_Session_NoArgs(t *testing.T) {
	var buf bytes.Buffer
	err := RunDebug([]string{"session"}, &buf)
	if err == nil {
		t.Fatal("expected error for missing session arg")
	}
}

func TestRunDebug_Frame_MissingFile(t *testing.T) {
	var buf bytes.Buffer
	err := RunDebug([]string{"frame", "/nonexistent/file.jsonl"}, &buf)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunDebug_UnknownSubcommand(t *testing.T) {
	var buf bytes.Buffer
	err := RunDebug([]string{"bogus"}, &buf)
	if err == nil {
		t.Fatal("expected error for unknown debug subcommand")
	}
}

func TestRunDebug_Component_DefaultFile(t *testing.T) {
	var buf bytes.Buffer
	// May succeed or fail depending on whether /tmp/djinn-frames.jsonl exists.
	// Just verify no panic.
	RunDebug([]string{"panels"}, &buf) //nolint:errcheck
}

func TestStripANSI(t *testing.T) {
	result := stripANSI("\x1b[31mhello\x1b[0m")
	if result != "hello" {
		t.Fatalf("stripANSI = %q, want hello", result)
	}
}

func TestStripANSI_NoEscape(t *testing.T) {
	result := stripANSI("plain text")
	if result != "plain text" {
		t.Fatalf("stripANSI = %q", result)
	}
}

func TestHubSocketExists_NoSocket(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, ok := HubSocketExists()
	if ok {
		t.Fatal("should not find hub socket in temp dir")
	}
}

func TestConnectToHub_BadPath(t *testing.T) {
	_, err := ConnectToHub("/nonexistent/socket.sock")
	if err == nil {
		t.Fatal("expected error for bad socket path")
	}
}

func TestConnectToHubAsBackend_BadPath(t *testing.T) {
	_, err := ConnectToHubAsBackend("/nonexistent/socket.sock")
	if err == nil {
		t.Fatal("expected error for bad socket path")
	}
}

func TestFindMostRecentJSONL_EmptyDir(t *testing.T) {
	result := findMostRecentJSONL(t.TempDir())
	if result != "" {
		t.Fatalf("empty dir should return empty, got %q", result)
	}
}

func TestFindMostRecentJSONL_WithFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "b.jsonl"), []byte("{}"), 0644)
	result := findMostRecentJSONL(dir)
	if result == "" {
		t.Fatal("should find a jsonl file")
	}
}

func TestPickPlaceholderFile_EmptyDirs(t *testing.T) {
	// Import the function via repl package — can't test directly from app.
	// Just verify no panic with empty dirs.
}

func TestDriverConstants(t *testing.T) {
	if DriverClaude != "claude" {
		t.Fatal("DriverClaude")
	}
	if DriverCursor != "cursor" {
		t.Fatal("DriverCursor")
	}
	if DriverOllama != "ollama" {
		t.Fatal("DriverOllama")
	}
}
