package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscover_ProjectLocal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: auto\n"), 0644)
	paths := Discover(dir)
	if len(paths) == 0 {
		t.Fatal("should find project config")
	}
	found := false
	for _, p := range paths {
		if strings.HasSuffix(p, "djinn.yaml") {
			found = true
		}
	}
	if !found {
		t.Fatalf("paths = %v, missing djinn.yaml", paths)
	}
}

func TestDiscover_EnvVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	os.WriteFile(path, []byte("mode: plan\n"), 0644)
	t.Setenv(EnvConfigVar, path)
	paths := Discover(dir)
	found := false
	for _, p := range paths {
		if p == path {
			found = true
		}
	}
	if !found {
		t.Fatalf("env var config not found in %v", paths)
	}
}

func TestDiscover_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	paths := Discover(dir)
	for _, p := range paths {
		if strings.Contains(p, "djinn.yaml") && strings.HasPrefix(p, dir) {
			t.Fatal("phantom project config")
		}
	}
}

func TestLoadAll_MergesInOrder(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: plan\n"), 0644)

	r := NewRegistry()
	mc := &ModeConfig{Mode: "agent"}
	r.Register(mc)
	if err := LoadAll(r, dir, ""); err != nil {
		t.Fatal(err)
	}
	if mc.Mode != "plan" {
		t.Fatalf("mode = %q, want plan", mc.Mode)
	}
}

func TestLoadAll_ExplicitOverrides(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: plan\n"), 0644)
	explicit := filepath.Join(dir, "override.yaml")
	os.WriteFile(explicit, []byte("mode: auto\n"), 0644)

	r := NewRegistry()
	mc := &ModeConfig{Mode: "agent"}
	r.Register(mc)
	if err := LoadAll(r, dir, explicit); err != nil {
		t.Fatal(err)
	}
	if mc.Mode != "auto" {
		t.Fatalf("mode = %q, want auto (explicit override)", mc.Mode)
	}
}

func TestLoadAll_ExplicitNotFound(t *testing.T) {
	r := NewRegistry()
	err := LoadAll(r, t.TempDir(), "/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("should error for missing explicit file")
	}
}

func TestLoadAll_NoFiles(t *testing.T) {
	r := NewRegistry()
	mc := &ModeConfig{Mode: "agent"}
	r.Register(mc)
	if err := LoadAll(r, t.TempDir(), ""); err != nil {
		t.Fatal(err)
	}
	if mc.Mode != "agent" {
		t.Fatal("should keep default when no files found")
	}
}
