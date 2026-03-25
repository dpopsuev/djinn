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

func TestDefaultConfig_ExecutorHasAllCapabilities(t *testing.T) {
	cfg := DefaultConfig()
	roles := cfg.RoleMap()
	exec, ok := roles["executor"]
	if !ok {
		t.Fatal("executor role not found")
	}
	if len(exec.ToolCapabilities) < 5 {
		t.Fatalf("executor should have many capabilities, got %d", len(exec.ToolCapabilities))
	}
}

func TestDefaultConfig_AuditorHasNoCodeCapabilities(t *testing.T) {
	cfg := DefaultConfig()
	roles := cfg.RoleMap()
	aud := roles["auditor"]
	for _, s := range aud.ToolCapabilities {
		if s == "FileEditing" || s == "ShellExecution" {
			t.Fatalf("auditor should NOT have %s capability", s)
		}
	}
}

func TestDefaultConfig_EmptyCapabilitiesIsNothing(t *testing.T) {
	r := Role{Name: "empty", ToolCapabilities: []string{}}
	if len(r.ToolCapabilities) != 0 {
		t.Fatal("empty capabilities should mean zero access")
	}
}

func TestDefaultConfig_CapabilitiesExist(t *testing.T) {
	cfg := DefaultConfig()
	caps := cfg.ToolCapabilityMap()
	if len(caps) < 10 {
		t.Fatalf("expected 10+ capabilities, got %d", len(caps))
	}
	if _, ok := caps["WorkTracking"]; !ok {
		t.Fatal("missing WorkTracking capability")
	}
	if _, ok := caps["FileEditing"]; !ok {
		t.Fatal("missing FileEditing capability")
	}
}

func TestDefaultConfig_ToolCategoriesExist(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.ToolCategories) != 8 {
		t.Fatalf("expected 8 DevOps categories, got %d", len(cfg.ToolCategories))
	}
	if len(cfg.ToolCategories["plan"]) < 3 {
		t.Fatal("plan category should have 3+ capabilities")
	}
	if len(cfg.ToolCategories["build"]) != 1 {
		t.Fatal("build category should have ShellExecution")
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
    tool_capabilities: [FileEditing, ShellExecution]
  - name: reviewer
    prompt: "Inline prompt text here"
    mode: plan
    tool_capabilities: [WorkTracking]
tool_capabilities:
  - name: FileEditing
    backend: builtin
    tools: [Read, Write, Edit]
  - name: ShellExecution
    backend: builtin
    tools: [Bash]
  - name: WorkTracking
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
	if len(custom.ToolCapabilities) != 2 {
		t.Fatalf("capabilities = %d", len(custom.ToolCapabilities))
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
    tool_capabilities: []
`
	cfgFile := filepath.Join(dir, "staff.yaml")
	os.WriteFile(cfgFile, []byte(yamlContent), 0644) //nolint:errcheck

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	r := cfg.RoleMap()["test"]
	if r.Prompt != "nonexistent-file.md" {
		t.Fatalf("prompt fallback = %q", r.Prompt)
	}
}
