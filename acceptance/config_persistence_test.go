package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/app"
	"github.com/dpopsuev/djinn/config"

	"gopkg.in/yaml.v3"
)

func TestConfig_RegistryDumpKeys(t *testing.T) {
	r := config.NewRegistry()
	r.Register(&config.ModeConfig{Mode: "agent"})
	r.Register(&config.DriverConfigurable{Name: "claude", Model: "opus"})
	r.Register(&config.SessionConfigurable{MaxTurns: 20})

	dump := r.Dump()
	for _, key := range []string{"mode", "driver", "session"} {
		if _, ok := dump[key]; !ok {
			t.Fatalf("missing key %q in dump", key)
		}
	}
}

func TestConfig_LoadInvalidReturnsError(t *testing.T) {
	r := config.NewRegistry()
	r.Register(&config.ModeConfig{Mode: "agent"})

	err := r.Load(map[string]any{"mode": 42})
	if err == nil {
		t.Fatal("expected error for invalid mode type")
	}
}

func TestConfig_DumpProducesValidYAML(t *testing.T) {
	var buf strings.Builder
	err := app.RunConfig([]string{"dump"}, &buf)
	if err != nil {
		t.Fatalf("config dump: %v", err)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\n%s", err, buf.String())
	}
}

func TestConfig_FileOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: auto\n"), 0o644)

	r := config.NewRegistry()
	mc := &config.ModeConfig{Mode: "agent"}
	r.Register(mc)
	config.LoadAll(r, dir, "")

	if mc.Mode != "auto" {
		t.Fatalf("mode = %q, want auto (from file)", mc.Mode)
	}
}

func TestConfig_CLIOverridesFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: plan\n"), 0o644)
	explicit := filepath.Join(dir, "override.yaml")
	os.WriteFile(explicit, []byte("mode: auto\n"), 0o644)

	r := config.NewRegistry()
	mc := &config.ModeConfig{Mode: "agent"}
	r.Register(mc)
	config.LoadAll(r, dir, explicit)

	if mc.Mode != "auto" {
		t.Fatalf("explicit override: mode = %q, want auto", mc.Mode)
	}
}

func TestConfig_NoFileUsesDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := config.NewRegistry()
	mc := &config.ModeConfig{Mode: "agent"}
	r.Register(mc)
	config.LoadAll(r, t.TempDir(), "")

	if mc.Mode != "agent" {
		t.Fatalf("mode = %q, want agent (default)", mc.Mode)
	}
}

func TestConfig_DumpLoadRoundtrip(t *testing.T) {
	r1 := config.NewRegistry()
	r1.Register(&config.ModeConfig{Mode: "auto"})
	r1.Register(&config.SessionConfigurable{MaxTurns: 42})
	data, err := r1.DumpYAML()
	if err != nil {
		t.Fatal(err)
	}

	r2 := config.NewRegistry()
	mc := &config.ModeConfig{Mode: "agent"}
	sc := &config.SessionConfigurable{MaxTurns: 20}
	r2.Register(mc)
	r2.Register(sc)
	if err := r2.LoadYAML(data); err != nil {
		t.Fatal(err)
	}

	if mc.Mode != "auto" {
		t.Fatalf("roundtrip mode = %q", mc.Mode)
	}
	if sc.MaxTurns != 42 {
		t.Fatalf("roundtrip max_turns = %d", sc.MaxTurns)
	}
}

func TestConfig_PartialApply(t *testing.T) {
	r := config.NewRegistry()
	sc := &config.SessionConfigurable{MaxTurns: 20, AutoApprove: false}
	r.Register(sc)

	r.LoadYAML([]byte("session:\n  max_turns: 50\n"))

	if sc.MaxTurns != 50 {
		t.Fatalf("max_turns = %d, want 50", sc.MaxTurns)
	}
	if sc.AutoApprove {
		t.Fatal("auto_approve should remain false (not in partial)")
	}
}
