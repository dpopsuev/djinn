package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDiscover_ProjectLocal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: auto\n"), 0o644)
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
	os.WriteFile(path, []byte("mode: plan\n"), 0o644)
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
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: plan\n"), 0o644)

	r := NewRegistry()
	mc := &ModeConfig{Mode: "agent"}
	r.Register(mc)
	if err := LoadAll(r, dir, ""); err != nil {
		t.Fatal(err)
	}
	if mc.Mode != ModePlan {
		t.Fatalf("mode = %q, want plan", mc.Mode)
	}
}

func TestLoadAll_ExplicitOverrides(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: plan\n"), 0o644)
	explicit := filepath.Join(dir, "override.yaml")
	os.WriteFile(explicit, []byte("mode: auto\n"), 0o644)

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

// --- TSK-616: Config parent-directory walk tests ---

func TestDiscover_FindsInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte("mode: auto\n"), 0o644)

	paths := Discover(dir)
	found := false
	for _, p := range paths {
		if p == filepath.Join(dir, "djinn.yaml") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Discover(%q) = %v, missing djinn.yaml in current dir", dir, paths)
	}
}

func TestDiscover_WalksParentDirs(t *testing.T) {
	// Create a nested directory structure:
	//   root/djinn.yaml
	//   root/a/b/c/djinn.yaml
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(deep, 0o755)

	rootConfig := filepath.Join(root, "djinn.yaml")
	deepConfig := filepath.Join(deep, "djinn.yaml")
	os.WriteFile(rootConfig, []byte("mode: plan\n"), 0o644)
	os.WriteFile(deepConfig, []byte("mode: auto\n"), 0o644)

	paths := Discover(deep)

	// Both should be found.
	rootFound := slices.Contains(paths, rootConfig)
	deepFound := slices.Contains(paths, deepConfig)

	if !rootFound {
		t.Errorf("parent config %q not found in %v", rootConfig, paths)
	}
	if !deepFound {
		t.Errorf("deep config %q not found in %v", deepConfig, paths)
	}
}

func TestDiscover_StopsAtRoot(t *testing.T) {
	// Use a temp dir deep enough that we know the walk terminates.
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c", "d", "e")
	os.MkdirAll(deep, 0o755)

	// Only put config in the deepest dir.
	deepConfig := filepath.Join(deep, "djinn.yaml")
	os.WriteFile(deepConfig, []byte("mode: auto\n"), 0o644)

	paths := Discover(deep)

	// Should find the deep config but not anything from outside root.
	for _, p := range paths {
		if strings.HasSuffix(p, "djinn.yaml") && !strings.HasPrefix(p, root) {
			// This is a config found outside our temp tree (e.g., global).
			// That's fine — global is expected. Just verify no panic.
			continue
		}
	}
	if !slices.Contains(paths, deepConfig) {
		t.Fatalf("deep config not found in %v", paths)
	}
}

func TestDiscover_DeepestOverridesShallowest(t *testing.T) {
	// The returned order must be: global → shallowest → ... → deepest → env.
	// When LoadAll applies them in order, deeper overrides shallower.
	root := t.TempDir()
	mid := filepath.Join(root, "a")
	deep := filepath.Join(root, "a", "b")
	os.MkdirAll(deep, 0o755)

	rootConfig := filepath.Join(root, "djinn.yaml")
	midConfig := filepath.Join(mid, "djinn.yaml")
	deepConfig := filepath.Join(deep, "djinn.yaml")

	os.WriteFile(rootConfig, []byte("mode: plan\n"), 0o644)
	os.WriteFile(midConfig, []byte("mode: plan\n"), 0o644)
	os.WriteFile(deepConfig, []byte("mode: auto\n"), 0o644)

	paths := Discover(deep)

	// Find the indices of our three configs.
	rootIdx := slices.Index(paths, rootConfig)
	midIdx := slices.Index(paths, midConfig)
	deepIdx := slices.Index(paths, deepConfig)

	if rootIdx == -1 || midIdx == -1 || deepIdx == -1 {
		t.Fatalf("missing configs in paths %v\n  root=%d mid=%d deep=%d", paths, rootIdx, midIdx, deepIdx)
	}

	// Shallowest (root) must come before mid, and mid before deepest.
	if rootIdx >= midIdx {
		t.Errorf("root (%d) should come before mid (%d)", rootIdx, midIdx)
	}
	if midIdx >= deepIdx {
		t.Errorf("mid (%d) should come before deep (%d)", midIdx, deepIdx)
	}

	// Verify LoadAll applies deepest last (overriding shallowest).
	r := NewRegistry()
	mc := &ModeConfig{Mode: "agent"}
	r.Register(mc)
	if err := LoadAll(r, deep, ""); err != nil {
		t.Fatal(err)
	}
	// Deep config has mode: auto, which should override root's mode: plan.
	if mc.Mode != "auto" {
		t.Fatalf("mode = %q, want auto (deepest should override shallowest)", mc.Mode)
	}
}
