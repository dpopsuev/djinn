package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectContext_FindsClaudeMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Claude Instructions"), 0644)

	ctx := LoadProjectContext(dir)
	if ctx.ClaudeMD == "" {
		t.Fatal("ClaudeMD should be loaded")
	}
	if !strings.Contains(ctx.ClaudeMD, "Claude Instructions") {
		t.Fatalf("ClaudeMD = %q", ctx.ClaudeMD)
	}
}

func TestLoadProjectContext_FindsClaudeMDInSubdir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0755)
	os.WriteFile(filepath.Join(dir, ".claude", "CLAUDE.md"), []byte("# Nested"), 0644)

	ctx := LoadProjectContext(dir)
	if !strings.Contains(ctx.ClaudeMD, "Nested") {
		t.Fatalf("ClaudeMD from .claude/ = %q", ctx.ClaudeMD)
	}
}

func TestLoadProjectContext_FindsAgentsMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Codex Instructions"), 0644)

	ctx := LoadProjectContext(dir)
	if !strings.Contains(ctx.AgentsMD, "Codex Instructions") {
		t.Fatalf("AgentsMD = %q", ctx.AgentsMD)
	}
}

func TestLoadProjectContext_FindsGeminiMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "GEMINI.md"), []byte("# Gemini Instructions"), 0644)

	ctx := LoadProjectContext(dir)
	if !strings.Contains(ctx.GeminiMD, "Gemini Instructions") {
		t.Fatalf("GeminiMD = %q", ctx.GeminiMD)
	}
}

func TestLoadProjectContext_FindsCursorRules(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".cursorrules"), []byte("use typescript"), 0644)

	ctx := LoadProjectContext(dir)
	if !strings.Contains(ctx.CursorRules, "use typescript") {
		t.Fatalf("CursorRules = %q", ctx.CursorRules)
	}
}

func TestLoadProjectContext_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx := LoadProjectContext(dir)
	if ctx.ClaudeMD != "" || ctx.AgentsMD != "" || ctx.GeminiMD != "" {
		t.Fatal("empty dir should produce empty context")
	}
	if ctx.WorkDir != dir {
		t.Fatalf("WorkDir = %q, want %q", ctx.WorkDir, dir)
	}
}

func TestLoadProjectContext_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("claude"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("codex"), 0644)
	os.WriteFile(filepath.Join(dir, "GEMINI.md"), []byte("gemini"), 0644)

	ctx := LoadProjectContext(dir)
	if ctx.ClaudeMD != "claude" {
		t.Fatalf("ClaudeMD = %q", ctx.ClaudeMD)
	}
	if ctx.AgentsMD != "codex" {
		t.Fatalf("AgentsMD = %q", ctx.AgentsMD)
	}
	if ctx.GeminiMD != "gemini" {
		t.Fatalf("GeminiMD = %q", ctx.GeminiMD)
	}
}

// --- Upward walk tests ---

func TestLoadProjectContext_WalksUp(t *testing.T) {
	// Create: parent/CLAUDE.md, parent/child/
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	os.MkdirAll(child, 0755)
	os.WriteFile(filepath.Join(parent, "CLAUDE.md"), []byte("found from parent"), 0644)

	ctx := LoadProjectContext(child)
	if ctx.ClaudeMD != "found from parent" {
		t.Fatalf("should find CLAUDE.md from parent dir, got %q", ctx.ClaudeMD)
	}
}

func TestLoadProjectContext_WalksUpTwoLevels(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("found two levels up"), 0644)

	ctx := LoadProjectContext(deep)
	if ctx.ClaudeMD != "found two levels up" {
		t.Fatalf("should walk up two levels, got %q", ctx.ClaudeMD)
	}
}

func TestLoadProjectContext_ClosestWins(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	os.MkdirAll(child, 0755)
	os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("root version"), 0644)
	os.WriteFile(filepath.Join(child, "CLAUDE.md"), []byte("child version"), 0644)

	ctx := LoadProjectContext(child)
	if ctx.ClaudeMD != "child version" {
		t.Fatalf("closest CLAUDE.md should win, got %q", ctx.ClaudeMD)
	}
}

func TestLoadProjectContext_MultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "CLAUDE.md"), []byte("from dir1"), 0644)
	os.WriteFile(filepath.Join(dir2, "AGENTS.md"), []byte("from dir2"), 0644)

	ctx := LoadProjectContext(dir1, dir2)
	if ctx.ClaudeMD != "from dir1" {
		t.Fatalf("ClaudeMD = %q", ctx.ClaudeMD)
	}
	if ctx.AgentsMD != "from dir2" {
		t.Fatalf("AgentsMD = %q", ctx.AgentsMD)
	}
	if len(ctx.WorkDirs) != 2 {
		t.Fatalf("WorkDirs = %d, want 2", len(ctx.WorkDirs))
	}
}

func TestLoadProjectContext_NoDirs(t *testing.T) {
	ctx := LoadProjectContext()
	if ctx.WorkDir != "" {
		t.Fatal("no dirs should produce empty context")
	}
}

// --- MEMORY.md tests ---

func TestDiscoverMemory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := filepath.Join(home, "Workspace", "djinn")
	os.MkdirAll(workDir, 0755)

	// Create Claude memory path
	slug := strings.ReplaceAll(workDir, "/", "-")
	memDir := filepath.Join(home, ".claude", "projects", slug, "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("# Project Memory\nDjinn is an agent runtime."), 0644)

	content := discoverMemory([]string{workDir})
	if !strings.Contains(content, "Project Memory") {
		t.Fatalf("MEMORY.md not found, got %q", content)
	}
}

func TestDiscoverMemory_NotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	content := discoverMemory([]string{"/nonexistent/path"})
	if content != "" {
		t.Fatal("should return empty for missing memory")
	}
}

func TestBuildSystemPrompt_IncludesMemory(t *testing.T) {
	ctx := ProjectContext{
		ClaudeMD: "project rules",
		MemoryMD: "memory context",
	}
	prompt := BuildSystemPrompt(ctx, "")
	if !strings.Contains(prompt, "memory context") {
		t.Fatal("prompt should include MEMORY.md")
	}
}

// --- Existing BuildSystemPrompt tests ---

func TestBuildSystemPrompt_ClaudeMDOnly(t *testing.T) {
	ctx := ProjectContext{ClaudeMD: "# Project\nUse Go."}
	prompt := BuildSystemPrompt(ctx, "")
	if !strings.Contains(prompt, "Use Go") {
		t.Fatalf("prompt missing CLAUDE.md content: %q", prompt)
	}
}

func TestBuildSystemPrompt_MergesAll(t *testing.T) {
	ctx := ProjectContext{
		ClaudeMD: "claude rules",
		AgentsMD: "codex rules",
		GeminiMD: "gemini rules",
	}
	prompt := BuildSystemPrompt(ctx, "user override")
	if !strings.Contains(prompt, "claude rules") {
		t.Fatal("missing claude")
	}
	if !strings.Contains(prompt, "codex rules") {
		t.Fatal("missing codex")
	}
	if !strings.Contains(prompt, "gemini rules") {
		t.Fatal("missing gemini")
	}
	if !strings.Contains(prompt, "user override") {
		t.Fatal("missing user override")
	}
}

func TestBuildSystemPrompt_UserOverrideAppended(t *testing.T) {
	ctx := ProjectContext{ClaudeMD: "base"}
	prompt := BuildSystemPrompt(ctx, "extra")
	baseIdx := strings.Index(prompt, "base")
	extraIdx := strings.Index(prompt, "extra")
	if extraIdx < baseIdx {
		t.Fatal("user override should come after project context")
	}
}

func TestBuildSystemPrompt_EmptyContext(t *testing.T) {
	ctx := ProjectContext{}
	prompt := BuildSystemPrompt(ctx, "just user")
	if !strings.Contains(prompt, "just user") {
		t.Fatalf("prompt = %q", prompt)
	}
}
