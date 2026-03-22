package acceptance

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/app"
	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/workspace"
)

// --- Upward walk (from DJA-SPC-5 scenario 1) ---

func TestWorkspace_UpwardWalkFindsParent(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "src", "agent")
	os.MkdirAll(child, 0755)
	os.WriteFile(filepath.Join(parent, "CLAUDE.md"), []byte("project rules"), 0644)

	ctx := djinnctx.LoadProjectContext(child)
	if ctx.ClaudeMD != "project rules" {
		t.Fatalf("should find CLAUDE.md from parent, got %q", ctx.ClaudeMD)
	}
}

func TestWorkspace_WalkStopsAtHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	subdir := filepath.Join(home, "projects", "test")
	os.MkdirAll(subdir, 0755)

	ctx := djinnctx.LoadProjectContext(subdir)
	// Should not find anything above home
	if ctx.ClaudeMD != "" {
		t.Fatal("walk should not escape above $HOME")
	}
}

func TestWorkspace_NoFileNoError(t *testing.T) {
	ctx := djinnctx.LoadProjectContext(t.TempDir())
	if ctx.ClaudeMD != "" || ctx.AgentsMD != "" || ctx.GeminiMD != "" {
		t.Fatal("empty dir should produce empty context")
	}
}

func TestWorkspace_MultiDirMerge(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "CLAUDE.md"), []byte("claude from dir1"), 0644)
	os.WriteFile(filepath.Join(dir2, "AGENTS.md"), []byte("agents from dir2"), 0644)

	ctx := djinnctx.LoadProjectContext(dir1, dir2)
	if ctx.ClaudeMD != "claude from dir1" {
		t.Fatalf("ClaudeMD = %q", ctx.ClaudeMD)
	}
	if ctx.AgentsMD != "agents from dir2" {
		t.Fatalf("AgentsMD = %q", ctx.AgentsMD)
	}
}

// --- MEMORY.md (from DJA-SPC-5 scenario 3) ---

func TestWorkspace_MemoryFromClaudePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := filepath.Join(home, "Workspace", "djinn")
	os.MkdirAll(workDir, 0755)

	slug := strings.ReplaceAll(workDir, "/", "-")
	memDir := filepath.Join(home, ".claude", "projects", slug, "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("# Memory\nDjinn context"), 0644)

	ctx := djinnctx.LoadProjectContext(workDir)
	if !strings.Contains(ctx.MemoryMD, "Memory") {
		t.Fatalf("MEMORY.md not discovered, got %q", ctx.MemoryMD)
	}
}

func TestWorkspace_MultiMemory_AllReposLoaded(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create two repos with their own Claude memory
	djinnDir := filepath.Join(home, "Workspace", "djinn")
	misbahDir := filepath.Join(home, "Workspace", "misbah")
	os.MkdirAll(djinnDir, 0755)
	os.MkdirAll(misbahDir, 0755)

	// Create MEMORY.md for each
	for _, dir := range []string{djinnDir, misbahDir} {
		slug := strings.ReplaceAll(dir, "/", "-")
		memDir := filepath.Join(home, ".claude", "projects", slug, "memory")
		os.MkdirAll(memDir, 0755)
		name := filepath.Base(dir)
		os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("# "+name+" Memory\n"+name+" is important."), 0644)
	}

	ctx := djinnctx.LoadProjectContext(djinnDir, misbahDir)

	if !strings.Contains(ctx.MemoryMD, "djinn Memory") {
		t.Fatalf("missing djinn memory in: %q", ctx.MemoryMD)
	}
	if !strings.Contains(ctx.MemoryMD, "misbah Memory") {
		t.Fatalf("missing misbah memory in: %q", ctx.MemoryMD)
	}
}

func TestWorkspace_NoMemoryNoError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := djinnctx.LoadProjectContext(t.TempDir())
	if ctx.MemoryMD != "" {
		t.Fatal("missing memory should be empty")
	}
}

func TestWorkspace_BuildPromptIncludesMemory(t *testing.T) {
	ctx := djinnctx.ProjectContext{
		ClaudeMD: "project rules",
		MemoryMD: "remembered context",
	}
	prompt := djinnctx.BuildSystemPrompt(ctx, "user system")
	if !strings.Contains(prompt, "remembered context") {
		t.Fatal("system prompt should include MEMORY.md")
	}
}

// --- Workspace manifest (from DJA-SPC-5 rewrite) ---

func TestWorkspace_LoadManifestFile(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "ws.yaml")
	os.WriteFile(manifest, []byte(`
name: test-project
repos:
  - path: /home/user/project
    role: primary
  - path: /home/user/lib
    role: dependency
driver: claude
model: claude-opus-4-6
`), 0644)

	ws, err := workspace.Load(manifest)
	if err != nil {
		t.Fatalf("Load manifest: %v", err)
	}
	if ws.Name != "test-project" {
		t.Fatalf("name = %q", ws.Name)
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("repos = %d", len(ws.Repos))
	}
	if ws.Driver != "claude" {
		t.Fatalf("driver = %q", ws.Driver)
	}
}

func TestWorkspace_LoadByName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ws := &workspace.Workspace{
		Name:  "named-ws",
		Repos: []workspace.Repo{{Path: "/proj", Role: "primary"}},
	}
	workspace.Save(ws)

	loaded, err := workspace.Load("named-ws")
	if err != nil {
		t.Fatalf("Load by name: %v", err)
	}
	if loaded.Name != "named-ws" {
		t.Fatalf("name = %q", loaded.Name)
	}
}

func TestWorkspace_SaveAndList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workspace.Save(&workspace.Workspace{Name: "a", Repos: []workspace.Repo{{Path: "/a", Role: "primary"}}})
	workspace.Save(&workspace.Workspace{Name: "b", Repos: []workspace.Repo{{Path: "/b", Role: "primary"}}})

	list, err := workspace.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %d, want 2", len(list))
	}
}

func TestWorkspace_Ephemeral(t *testing.T) {
	ws := workspace.Ephemeral("/tmp/scratch")
	if ws.Name != "" {
		t.Fatal("ephemeral should have no name")
	}
	if ws.PrimaryPath() != "/tmp/scratch" {
		t.Fatalf("primary = %q", ws.PrimaryPath())
	}
}

func TestWorkspace_SessionPersistsWorkspace(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	sess := session.New("test", "model", "/workspace")
	sess.Workspace = "my-project"
	store.Save(sess)

	loaded, _ := store.Load("test")
	if loaded.Workspace != "my-project" {
		t.Fatalf("workspace = %q, want my-project", loaded.Workspace)
	}
}

// --- CLI workspace subcommand ---

func TestWorkspace_CLIList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var buf bytes.Buffer
	app.RunWorkspace([]string{"list"}, &buf)
	if !strings.Contains(buf.String(), "no workspaces") {
		t.Fatalf("empty list should say no workspaces: %q", buf.String())
	}
}

func TestWorkspace_CLICreate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var buf bytes.Buffer
	err := app.RunWorkspace([]string{"create", "test-ws", "--repo", "/home/user/proj"}, &buf)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(buf.String(), "test-ws") {
		t.Fatalf("output = %q", buf.String())
	}

	// Verify it was saved
	ws, err := workspace.Load("test-ws")
	if err != nil {
		t.Fatalf("Load after create: %v", err)
	}
	if len(ws.Repos) != 1 {
		t.Fatalf("repos = %d", len(ws.Repos))
	}
}
