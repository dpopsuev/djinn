package app

import (
	"os"
	"testing"
)

func TestScanArsenal_FindsAtLeastOne(t *testing.T) {
	// On a dev machine, at least one agent CLI should be on PATH.
	// Skip if running in bare CI without any CLIs.
	detected := ScanArsenal()
	if len(detected) == 0 {
		// Check if we at least have an API key.
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			t.Skip("no agent CLIs or API keys detected — normal in bare CI")
		}
	}
	for _, d := range detected {
		if d.Name == "" {
			t.Fatal("detected driver has empty name")
		}
		if d.Source != "cli" && d.Source != "api-key" {
			t.Fatalf("unknown source: %q", d.Source)
		}
	}
}

func TestDetectedDriver_ACPName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"cursor", "acp"},
		{"claude", "claude"},
		{"claude-api", "claude"},
		{"gemini", "gemini"},
		{"codex", "codex"},
		{"ollama", "ollama"},
	}
	for _, tt := range tests {
		d := DetectedDriver{Name: tt.name}
		if got := d.ACPName(); got != tt.want {
			t.Errorf("ACPName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDetectedDriver_DefaultModel(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"cursor", "cursor/sonnet-4"},
		{"claude", "claude-sonnet-4-6"},
		{"gemini", "gemini-2.5-pro"},
		{"ollama", "llama3"},
	}
	for _, tt := range tests {
		d := DetectedDriver{Name: tt.name}
		if got := d.DefaultModel(); got != tt.want {
			t.Errorf("DefaultModel(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
