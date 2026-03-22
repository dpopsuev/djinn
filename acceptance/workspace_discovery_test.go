package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	djinnctx "github.com/dpopsuev/djinn/context"
	"github.com/dpopsuev/djinn/session"
)

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
	// Put CLAUDE.md ABOVE home — should NOT be found
	aboveHome := filepath.Dir(home)
	os.WriteFile(filepath.Join(aboveHome, "CLAUDE.md"), []byte("above home"), 0644)
	defer os.Remove(filepath.Join(aboveHome, "CLAUDE.md"))

	subdir := filepath.Join(home, "projects", "test")
	os.MkdirAll(subdir, 0755)

	ctx := djinnctx.LoadProjectContext(subdir)
	if ctx.ClaudeMD == "above home" {
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

func TestWorkspace_NoMemoryNoError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := djinnctx.LoadProjectContext(t.TempDir())
	if ctx.MemoryMD != "" {
		t.Fatal("missing memory should be empty, not error")
	}
}

func TestWorkspace_SessionPersistsWorkDirs(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	sess := session.New("test", "model", "/workspace")
	sess.WorkDirs = []string{"/path/one", "/path/two"}
	store.Save(sess)

	loaded, err := store.Load("test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.WorkDirs) != 2 {
		t.Fatalf("WorkDirs = %v", loaded.WorkDirs)
	}
	if loaded.WorkDirs[0] != "/path/one" {
		t.Fatalf("WorkDirs[0] = %q", loaded.WorkDirs[0])
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
