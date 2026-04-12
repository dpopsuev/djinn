package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.yaml")

	yaml := `scope: /djinn
mounts:
  djinn:
    host: /home/user/Workspace/djinn
    scope: operations
  troupe:
    host: /home/user/Workspace/troupe
    readonly: true
  mirage:
    host: /home/user/Workspace/mirage
    readonly: true
    scope: global
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Scope != "/djinn" {
		t.Fatalf("scope = %q, want /djinn", cfg.Scope)
	}
	if len(cfg.Mounts) != 3 {
		t.Fatalf("mounts = %d, want 3", len(cfg.Mounts))
	}

	djinn := cfg.Mounts["djinn"]
	if djinn.Host != "/home/user/Workspace/djinn" {
		t.Fatalf("djinn host = %q", djinn.Host)
	}
	if djinn.ReadOnly {
		t.Fatal("djinn should not be readonly")
	}

	troupe := cfg.Mounts["troupe"]
	if !troupe.ReadOnly {
		t.Fatal("troupe should be readonly")
	}

	mirage := cfg.Mounts["mirage"]
	if mirage.Scope != "global" {
		t.Fatalf("mirage scope = %q, want global", mirage.Scope)
	}
}

func TestSaveManifest_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.yaml")

	original := &ManifestConfig{
		Scope: "/djinn",
		Mounts: map[string]MountSpec{
			"djinn":  {Host: "/home/user/djinn", Scope: "operations"},
			"troupe": {Host: "/home/user/troupe", ReadOnly: true},
		},
	}

	if err := SaveManifest(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Scope != original.Scope {
		t.Fatalf("scope = %q, want %q", loaded.Scope, original.Scope)
	}
	if len(loaded.Mounts) != 2 {
		t.Fatalf("mounts = %d, want 2", len(loaded.Mounts))
	}
	if loaded.Mounts["djinn"].Host != "/home/user/djinn" {
		t.Fatalf("djinn host = %q", loaded.Mounts["djinn"].Host)
	}
}

func TestPopulateMountTable(t *testing.T) {
	cfg := &ManifestConfig{
		Scope: "/djinn",
		Mounts: map[string]MountSpec{
			"djinn":  {Host: "/home/user/djinn", Scope: "operations"},
			"troupe": {Host: "/home/user/troupe", ReadOnly: true, Scope: "global"},
		},
	}

	table := NewMountTable(nil)
	if err := PopulateMountTable(cfg, table, nil); err != nil {
		t.Fatal(err)
	}

	mounts := table.List()
	if len(mounts) != 2 {
		t.Fatalf("mounts = %d, want 2", len(mounts))
	}

	// Translate works through populated table.
	host, err := table.Translate("/djinn/djinn/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if host != "/home/user/djinn/main.go" {
		t.Fatalf("host = %q", host)
	}
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	_, err := LoadManifest("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadManifest_BadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("{{{{invalid"), 0o644) //nolint:errcheck // test helper, error irrelevant

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for bad YAML")
	}
}
