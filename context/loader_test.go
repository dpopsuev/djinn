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

func TestBuildSystemPrompt_ClaudeMDOnly(t *testing.T) {
	ctx := ProjectContext{ClaudeMD: "# Project\nUse Go."}
	prompt := BuildSystemPrompt(ctx, "")

	if !strings.Contains(prompt, "Use Go") {
		t.Fatalf("prompt missing CLAUDE.md content: %q", prompt)
	}
}

func TestBuildSystemPrompt_MergesAll(t *testing.T) {
	ctx := ProjectContext{
		ClaudeMD:  "claude rules",
		AgentsMD:  "codex rules",
		GeminiMD:  "gemini rules",
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

	// User override should come after project context
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
