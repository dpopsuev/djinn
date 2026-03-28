package staff

import (
	"os"
	"path/filepath"
	"strings"
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
	os.WriteFile(promptFile, []byte("You are Custom Role."), 0o644) //nolint:errcheck // best-effort write

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
	os.WriteFile(cfgFile, []byte(yamlContent), 0o644) //nolint:errcheck // best-effort write

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
	os.WriteFile(cfgFile, []byte(yamlContent), 0o644) //nolint:errcheck // best-effort write

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	r := cfg.RoleMap()["test"]
	if r.Prompt != "nonexistent-file.md" {
		t.Fatalf("prompt fallback = %q", r.Prompt)
	}
}

func TestResolveToolNames(t *testing.T) {
	cfg := DefaultConfig()
	tools := cfg.ResolveToolNames([]string{"FileEditing", "ShellExecution"})

	want := map[string]bool{"Read": true, "Write": true, "Edit": true, "Bash": true}
	got := map[string]bool{}
	for _, t := range tools {
		got[t] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("missing tool %q", name)
		}
	}
}

func TestResolveToolNames_IncludesMCPPrefix(t *testing.T) {
	cfg := DefaultConfig()
	tools := cfg.ResolveToolNames([]string{"WorkTracking"})

	got := map[string]bool{}
	for _, t := range tools {
		got[t] = true
	}
	if !got["artifact"] {
		t.Error("missing raw tool name 'artifact'")
	}
	if !got["mcp__scribe__artifact"] {
		t.Error("missing MCP-prefixed 'mcp__scribe__artifact'")
	}
}

func TestResolveToolNames_UnknownCapability(t *testing.T) {
	cfg := DefaultConfig()
	tools := cfg.ResolveToolNames([]string{"NonExistent"})
	if len(tools) != 0 {
		t.Fatalf("unknown capability should resolve to 0 tools, got %d", len(tools))
	}
}

func TestValidate_DefaultConfigPasses(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestValidate_UnknownCapabilityInRole(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Roles = append(cfg.Roles, Role{
		Name:             "broken",
		ToolCapabilities: []string{"FakeCapability"},
	})
	err := cfg.Validate()
	if err == nil {
		t.Fatal("should fail on unknown capability")
	}
	if !strings.Contains(err.Error(), "FakeCapability") {
		t.Fatalf("error should mention FakeCapability: %v", err)
	}
}

func TestValidate_UnknownCapabilityInCategory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ToolCategories["test"] = []string{"GhostCapability"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("should fail on unknown capability in category")
	}
}

func TestMergeConfig_OverlayReplacesRole(t *testing.T) {
	base := DefaultConfig()
	overlay := &StaffConfig{
		Roles: []Role{
			{Name: "executor", Mode: "plan", ToolCapabilities: []string{"WorkTracking"}},
		},
	}
	merged := MergeConfig(base, overlay)

	exec := merged.RoleMap()["executor"]
	if exec.Mode != "plan" {
		t.Fatalf("executor mode = %q, want plan (overlay)", exec.Mode)
	}
	if len(exec.ToolCapabilities) != 1 || exec.ToolCapabilities[0] != "WorkTracking" {
		t.Fatalf("executor capabilities = %v, want [WorkTracking]", exec.ToolCapabilities)
	}

	// Other roles should be preserved from base.
	if _, ok := merged.RoleMap()["gensec"]; !ok {
		t.Fatal("gensec should be preserved from base")
	}
}

func TestMergeConfig_OverlayAddsRole(t *testing.T) {
	base := DefaultConfig()
	overlay := &StaffConfig{
		Roles: []Role{
			{Name: "newrole", Mode: "agent", ToolCapabilities: []string{"FileEditing"}},
		},
	}
	merged := MergeConfig(base, overlay)
	if _, ok := merged.RoleMap()["newrole"]; !ok {
		t.Fatal("new role should be added")
	}
}

func TestMergeConfig_OverlayReplacesCapability(t *testing.T) {
	base := DefaultConfig()
	overlay := &StaffConfig{
		ToolCapabilities: []ToolCapability{
			{Name: "FileEditing", Backend: "custom", Tools: []string{"CustomEdit"}},
		},
	}
	merged := MergeConfig(base, overlay)
	tc := merged.ToolCapabilityMap()["FileEditing"]
	if tc.Backend != "custom" {
		t.Fatalf("FileEditing backend = %q, want custom", tc.Backend)
	}
}

func TestLoadConfigChain_NoFiles(t *testing.T) {
	cfg := LoadConfigChain("/nonexistent/a.yaml", "/nonexistent/b.yaml")
	if cfg == nil {
		t.Fatal("should return defaults when no files exist")
	}
	if len(cfg.Roles) != 5 {
		t.Fatalf("should have 5 default roles, got %d", len(cfg.Roles))
	}
}
