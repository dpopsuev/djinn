package config

import (
	"testing"
)

func TestModeConfig_Roundtrip(t *testing.T) {
	c := &ModeConfig{Mode: "auto"}
	if c.ConfigKey() != "mode" {
		t.Fatalf("key = %q", c.ConfigKey())
	}
	snap := c.Snapshot()
	c2 := &ModeConfig{Mode: "agent"}
	if err := c2.Apply(snap); err != nil {
		t.Fatal(err)
	}
	if c2.Mode != "auto" {
		t.Fatalf("mode = %q, want auto", c2.Mode)
	}
}

func TestModeConfig_InvalidMode(t *testing.T) {
	c := &ModeConfig{}
	if err := c.Apply("yolo"); err == nil {
		t.Fatal("should reject invalid mode")
	}
}

func TestModeConfig_WrongType(t *testing.T) {
	c := &ModeConfig{}
	if err := c.Apply(42); err == nil {
		t.Fatal("should reject non-string")
	}
}

func TestDriverConfigurable_Roundtrip(t *testing.T) {
	c := &DriverConfigurable{Name: "claude", Model: "claude-opus-4-6"}
	if c.ConfigKey() != "driver" {
		t.Fatalf("key = %q", c.ConfigKey())
	}
	snap := c.Snapshot().(map[string]string)
	if snap["name"] != "claude" || snap["model"] != "claude-opus-4-6" {
		t.Fatalf("snap = %v", snap)
	}

	c2 := &DriverConfigurable{}
	if err := c2.Apply(map[string]any{"name": "ollama", "model": "qwen2.5"}); err != nil {
		t.Fatal(err)
	}
	if c2.Name != "ollama" || c2.Model != "qwen2.5" {
		t.Fatalf("c2 = %s/%s", c2.Name, c2.Model)
	}
}

func TestDriverConfigurable_NoSecrets(t *testing.T) {
	c := &DriverConfigurable{Name: "claude", Model: "opus"}
	snap := c.Snapshot().(map[string]string)
	for k := range snap {
		if k == "api_key" || k == "token" {
			t.Fatalf("snapshot contains secret: %q", k)
		}
	}
}

func TestSessionConfigurable_Roundtrip(t *testing.T) {
	c := &SessionConfigurable{MaxTurns: 50, AutoApprove: true, OutputMode: "chunked", NoPersist: true}
	if c.ConfigKey() != "session" {
		t.Fatalf("key = %q", c.ConfigKey())
	}

	c2 := &SessionConfigurable{}
	if err := c2.Apply(c.Snapshot()); err != nil {
		t.Fatal(err)
	}
	if c2.MaxTurns != 50 {
		t.Fatalf("max_turns = %d", c2.MaxTurns)
	}
	if !c2.AutoApprove {
		t.Fatal("auto_approve should be true")
	}
	if c2.OutputMode != "chunked" {
		t.Fatalf("output_mode = %q", c2.OutputMode)
	}
}

func TestSessionConfigurable_PartialApply(t *testing.T) {
	c := &SessionConfigurable{MaxTurns: 20, AutoApprove: false}
	if err := c.Apply(map[string]any{"max_turns": 50}); err != nil {
		t.Fatal(err)
	}
	if c.MaxTurns != 50 {
		t.Fatalf("max_turns = %d, want 50", c.MaxTurns)
	}
	if c.AutoApprove {
		t.Fatal("auto_approve should remain false")
	}
}

func TestToolsConfigurable_Roundtrip(t *testing.T) {
	c := &ToolsConfigurable{Enabled: []string{"Read", "Write", "Bash"}}
	if c.ConfigKey() != "tools" {
		t.Fatalf("key = %q", c.ConfigKey())
	}

	c2 := &ToolsConfigurable{}
	if err := c2.Apply(c.Snapshot()); err != nil {
		t.Fatal(err)
	}
	if len(c2.Enabled) != 3 {
		t.Fatalf("enabled = %v", c2.Enabled)
	}
}

func TestToolsConfigurable_FromYAMLTypes(t *testing.T) {
	// YAML unmarshals lists as []any, not []string
	c := &ToolsConfigurable{}
	if err := c.Apply(map[string]any{
		"enabled": []any{"Read", "Write"},
	}); err != nil {
		t.Fatal(err)
	}
	if len(c.Enabled) != 2 || c.Enabled[0] != "Read" {
		t.Fatalf("enabled = %v", c.Enabled)
	}
}
