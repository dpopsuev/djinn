package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateConfig_WritesValidYAML(t *testing.T) {
	dir := t.TempDir()
	drv := DetectedDriver{Name: "cursor"}

	if err := GenerateConfig(dir, drv); err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	path := filepath.Join(dir, "djinn.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: acp") {
		t.Fatal("should contain driver name")
	}
	if !strings.Contains(content, "model: cursor/sonnet-4") {
		t.Fatal("should contain default model")
	}
	if !strings.Contains(content, "mode: agent") {
		t.Fatal("should contain mode")
	}
}

func TestGenerateConfig_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "djinn.yaml")
	os.WriteFile(path, []byte("existing"), 0o644)

	err := GenerateConfig(dir, DetectedDriver{Name: "cursor"})
	if err == nil {
		t.Fatal("should error when djinn.yaml exists")
	}

	// Original content preserved.
	data, _ := os.ReadFile(path)
	if string(data) != "existing" {
		t.Fatal("should not overwrite existing config")
	}
}

func TestFriendlyNoDriverError_ContainsInstallURLs(t *testing.T) {
	msg := FriendlyNoDriverError()
	for _, keyword := range []string{"cursor", "claude", "gemini", "ollama", "ANTHROPIC_API_KEY"} {
		if !strings.Contains(msg, keyword) {
			t.Errorf("should mention %q", keyword)
		}
	}
}

func TestFriendlyDriverNotFoundError_KnownDriver(t *testing.T) {
	msg := FriendlyDriverNotFoundError("cursor")
	if !strings.Contains(msg, "cursor.com") {
		t.Fatal("should contain install URL for cursor")
	}
}

func TestFriendlyDriverNotFoundError_UnknownDriver(t *testing.T) {
	msg := FriendlyDriverNotFoundError("foobar")
	if !strings.Contains(msg, "djinn doctor") {
		t.Fatal("should suggest djinn doctor for unknown driver")
	}
}
