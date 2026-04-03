package config

import (
	"testing"
)

func TestParseHooks_ValidYAML(t *testing.T) {
	data := []byte(`
pre_tool_use:
  - command: "echo check"
    tools: ["Edit", "Write"]
  - command: "exit 0"
    tools: ["*"]
post_tool_use:
  - command: "echo done"
    tools: ["Bash"]
`)

	cfg, err := ParseHooks(data)
	if err != nil {
		t.Fatalf("ParseHooks() error = %v", err)
	}

	if len(cfg.PreToolUse) != 2 {
		t.Fatalf("PreToolUse = %d, want 2", len(cfg.PreToolUse))
	}
	if cfg.PreToolUse[0].Command != "echo check" {
		t.Fatalf("PreToolUse[0].Command = %q, want %q", cfg.PreToolUse[0].Command, "echo check")
	}
	if len(cfg.PreToolUse[0].Tools) != 2 {
		t.Fatalf("PreToolUse[0].Tools = %d, want 2", len(cfg.PreToolUse[0].Tools))
	}
	if cfg.PreToolUse[0].Tools[0] != "Edit" {
		t.Fatalf("PreToolUse[0].Tools[0] = %q, want %q", cfg.PreToolUse[0].Tools[0], "Edit")
	}

	if len(cfg.PostToolUse) != 1 {
		t.Fatalf("PostToolUse = %d, want 1", len(cfg.PostToolUse))
	}
	if cfg.PostToolUse[0].Command != "echo done" {
		t.Fatalf("PostToolUse[0].Command = %q, want %q", cfg.PostToolUse[0].Command, "echo done")
	}
}

func TestParseHooks_EmptyHooks(t *testing.T) {
	data := []byte(`
pre_tool_use: []
post_tool_use: []
`)

	cfg, err := ParseHooks(data)
	if err != nil {
		t.Fatalf("ParseHooks() error = %v", err)
	}

	if len(cfg.PreToolUse) != 0 {
		t.Fatalf("PreToolUse = %d, want 0", len(cfg.PreToolUse))
	}
	if len(cfg.PostToolUse) != 0 {
		t.Fatalf("PostToolUse = %d, want 0", len(cfg.PostToolUse))
	}
}

func TestParseHooks_EmptyYAML(t *testing.T) {
	cfg, err := ParseHooks([]byte(""))
	if err != nil {
		t.Fatalf("ParseHooks() error = %v", err)
	}

	if len(cfg.PreToolUse) != 0 {
		t.Fatalf("PreToolUse = %d, want 0", len(cfg.PreToolUse))
	}
	if len(cfg.PostToolUse) != 0 {
		t.Fatalf("PostToolUse = %d, want 0", len(cfg.PostToolUse))
	}
}

func TestParseHooks_InvalidYAML(t *testing.T) {
	_, err := ParseHooks([]byte("{{{invalid yaml"))
	if err == nil {
		t.Fatal("ParseHooks() expected error for invalid YAML")
	}
}
