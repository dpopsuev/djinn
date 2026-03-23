package staff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_HasAllRoles(t *testing.T) {
	cfg := DefaultConfig()
	names := cfg.RoleNames()
	want := []string{"auditor", "executor", "gensec", "inspector", "scheduler"}
	if len(names) != len(want) {
		t.Fatalf("roles = %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("role[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestDefaultConfig_ExecutorHasAllSlots(t *testing.T) {
	cfg := DefaultConfig()
	roles := cfg.RoleMap()
	exec, ok := roles["executor"]
	if !ok {
		t.Fatal("executor role not found")
	}
	if len(exec.Slots) < 5 {
		t.Fatalf("executor should have many slots, got %d", len(exec.Slots))
	}
}

func TestDefaultConfig_AuditorHasNoCodeSlots(t *testing.T) {
	cfg := DefaultConfig()
	roles := cfg.RoleMap()
	aud := roles["auditor"]
	for _, s := range aud.Slots {
		if s == "CodeEditor" || s == "Shell" {
			t.Fatalf("auditor should NOT have %s slot", s)
		}
	}
}

func TestDefaultConfig_EmptySlotsIsNothing(t *testing.T) {
	// Verify: empty slots means no access, not all access
	r := Role{Name: "empty", Slots: []string{}}
	if len(r.Slots) != 0 {
		t.Fatal("empty slots should mean zero access")
	}
}

func TestDefaultConfig_SlotsExist(t *testing.T) {
	cfg := DefaultConfig()
	slots := cfg.SlotMap()
	if len(slots) < 10 {
		t.Fatalf("expected 10+ slots, got %d", len(slots))
	}
	// Spot-check
	if _, ok := slots["WorkTracker"]; !ok {
		t.Fatal("missing WorkTracker slot")
	}
	if _, ok := slots["CodeEditor"]; !ok {
		t.Fatal("missing CodeEditor slot")
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "custom.md")
	os.WriteFile(promptFile, []byte("You are Custom Role."), 0644) //nolint:errcheck

	yamlContent := `
roles:
  - name: custom
    prompt: custom.md
    mode: agent
    slots: [CodeEditor, Shell]
  - name: reviewer
    prompt: "Inline prompt text here"
    mode: plan
    slots: [WorkTracker]
slots:
  - name: CodeEditor
    backend: builtin
    tools: [Read, Write, Edit]
  - name: Shell
    backend: builtin
    tools: [Bash]
  - name: WorkTracker
    backend: scribe
    tools: [artifact]
`
	cfgFile := filepath.Join(dir, "staff.yaml")
	os.WriteFile(cfgFile, []byte(yamlContent), 0644) //nolint:errcheck

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	roles := cfg.RoleMap()
	if len(roles) != 2 {
		t.Fatalf("roles = %d, want 2", len(roles))
	}

	custom := roles["custom"]
	if custom.Prompt != "You are Custom Role." {
		t.Fatalf("prompt not loaded from file: %q", custom.Prompt)
	}
	if custom.Mode != "agent" {
		t.Fatalf("mode = %q", custom.Mode)
	}
	if len(custom.Slots) != 2 {
		t.Fatalf("slots = %d", len(custom.Slots))
	}

	reviewer := roles["reviewer"]
	if reviewer.Prompt != "Inline prompt text here" {
		t.Fatalf("inline prompt = %q", reviewer.Prompt)
	}
}

func TestLoadConfig_PromptFallback(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
roles:
  - name: test
    prompt: nonexistent-file.md
    mode: plan
    slots: []
`
	cfgFile := filepath.Join(dir, "staff.yaml")
	os.WriteFile(cfgFile, []byte(yamlContent), 0644) //nolint:errcheck

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	r := cfg.RoleMap()["test"]
	// Should fall back to treating the string as-is
	if r.Prompt != "nonexistent-file.md" {
		t.Fatalf("prompt fallback = %q", r.Prompt)
	}
}
