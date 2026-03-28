package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "test.yaml")
	os.WriteFile(manifest, []byte(`
name: my-project
repos:
  - path: /home/user/project
    role: primary
  - path: /home/user/lib
    role: dependency
driver: claude
model: claude-opus-4-6
mode: agent
mcp:
  scribe:
    url: "http://localhost:8080/"
`), 0o644)

	ws, err := Load(manifest)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ws.Name != "my-project" {
		t.Fatalf("name = %q", ws.Name)
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("repos = %d", len(ws.Repos))
	}
	if ws.Driver != "claude" {
		t.Fatalf("driver = %q", ws.Driver)
	}
	if ws.MCP["scribe"].URL != "http://localhost:8080/" {
		t.Fatalf("mcp scribe = %v", ws.MCP["scribe"])
	}
}

func TestLoad_ByName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	wsDir := filepath.Join(home, ".config", "djinn", "workspaces")
	os.MkdirAll(wsDir, 0o755)
	os.WriteFile(filepath.Join(wsDir, "dev.yaml"), []byte(`
name: dev
repos:
  - path: /work
    role: primary
`), 0o644)

	ws, err := Load("dev")
	if err != nil {
		t.Fatalf("Load by name: %v", err)
	}
	if ws.Name != "dev" {
		t.Fatalf("name = %q", ws.Name)
	}
}

func TestLoad_NotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := Load("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_Empty(t *testing.T) {
	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSaveAndLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ws := &Workspace{
		Name:   "roundtrip",
		Repos:  []Repo{{Path: "/a", Role: RolePrimary}, {Path: "/b", Role: RoleDependency}},
		Driver: "ollama",
		Model:  "qwen2.5",
		Mode:   "auto",
	}
	if err := Save(ws); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("roundtrip")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Name != "roundtrip" {
		t.Fatalf("name = %q", loaded.Name)
	}
	if loaded.Driver != "ollama" {
		t.Fatalf("driver = %q", loaded.Driver)
	}
	if len(loaded.Repos) != 2 {
		t.Fatalf("repos = %d", len(loaded.Repos))
	}
}

func TestSave_NoName(t *testing.T) {
	err := Save(&Workspace{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	Save(&Workspace{Name: "alpha", Repos: []Repo{{Path: "/a", Role: RolePrimary}}})
	Save(&Workspace{Name: "beta", Repos: []Repo{{Path: "/b", Role: RolePrimary}, {Path: "/c", Role: RoleDependency}}})

	summaries, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("list = %d, want 2", len(summaries))
	}
}

func TestList_Empty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	summaries, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 0 {
		t.Fatalf("should be empty, got %d", len(summaries))
	}
}

func TestEphemeral(t *testing.T) {
	ws := Ephemeral("/tmp/scratch")
	if len(ws.Repos) != 1 {
		t.Fatalf("repos = %d", len(ws.Repos))
	}
	if ws.Repos[0].Path != "/tmp/scratch" {
		t.Fatalf("path = %q", ws.Repos[0].Path)
	}
	if ws.Repos[0].Role != RolePrimary {
		t.Fatalf("role = %q", ws.Repos[0].Role)
	}
}

func TestPaths(t *testing.T) {
	ws := &Workspace{Repos: []Repo{{Path: "/a"}, {Path: "/b"}, {Path: "/c"}}}
	paths := ws.Paths()
	if len(paths) != 3 || paths[0] != "/a" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestPrimaryPath(t *testing.T) {
	ws := &Workspace{Repos: []Repo{
		{Path: "/dep", Role: RoleDependency},
		{Path: "/main", Role: RolePrimary},
	}}
	if ws.PrimaryPath() != "/main" {
		t.Fatalf("primary = %q", ws.PrimaryPath())
	}
}

func TestPrimaryPath_FallsBackToFirst(t *testing.T) {
	ws := &Workspace{Repos: []Repo{{Path: "/only", Role: RoleReference}}}
	if ws.PrimaryPath() != "/only" {
		t.Fatalf("should fall back to first repo")
	}
}

func TestPrimaryPath_Empty(t *testing.T) {
	ws := &Workspace{}
	if ws.PrimaryPath() != "" {
		t.Fatal("empty workspace should return empty")
	}
}
