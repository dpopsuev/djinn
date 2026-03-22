// Package context discovers and loads project instruction files
// from multiple CLI conventions (Claude, Codex, Gemini, Cursor, Aider).
// Djinn reads ALL of them for seamless driver switching.
package context

import (
	"os"
	"path/filepath"
	"strings"
)

// Instruction file names by convention.
const (
	fileClaudeMD    = "CLAUDE.md"
	fileClaudeMDSub = ".claude/CLAUDE.md"
	fileAgentsMD    = "AGENTS.md"
	fileGeminiMD    = "GEMINI.md"
	fileCursorRules = ".cursorrules"
	fileCursorDir   = ".cursor/rules"
	fileAiderConf   = ".aider.conf.yml"
)

// ProjectContext holds discovered project instruction files.
type ProjectContext struct {
	ClaudeMD    string // CLAUDE.md or .claude/CLAUDE.md
	AgentsMD    string // AGENTS.md (Codex)
	GeminiMD    string // GEMINI.md
	CursorRules string // .cursorrules or .cursor/rules
	AiderConf   string // .aider.conf.yml
	WorkDir     string
}

// LoadProjectContext discovers and reads all project instruction files
// from the given directory. Missing files are silently skipped.
func LoadProjectContext(workdir string) ProjectContext {
	ctx := ProjectContext{WorkDir: workdir}

	// Claude: try root first, then .claude/ subdirectory
	ctx.ClaudeMD = readFileIfExists(filepath.Join(workdir, fileClaudeMD))
	if ctx.ClaudeMD == "" {
		ctx.ClaudeMD = readFileIfExists(filepath.Join(workdir, fileClaudeMDSub))
	}

	// Codex
	ctx.AgentsMD = readFileIfExists(filepath.Join(workdir, fileAgentsMD))

	// Gemini
	ctx.GeminiMD = readFileIfExists(filepath.Join(workdir, fileGeminiMD))

	// Cursor: try root first, then .cursor/ subdirectory
	ctx.CursorRules = readFileIfExists(filepath.Join(workdir, fileCursorRules))
	if ctx.CursorRules == "" {
		ctx.CursorRules = readFileIfExists(filepath.Join(workdir, fileCursorDir))
	}

	// Aider
	ctx.AiderConf = readFileIfExists(filepath.Join(workdir, fileAiderConf))

	return ctx
}

// BuildSystemPrompt assembles a system prompt from project context
// and an optional user-provided override. Project context comes first,
// user override is appended at the end.
func BuildSystemPrompt(ctx ProjectContext, userSystem string) string {
	var parts []string

	if ctx.ClaudeMD != "" {
		parts = append(parts, ctx.ClaudeMD)
	}
	if ctx.AgentsMD != "" {
		parts = append(parts, ctx.AgentsMD)
	}
	if ctx.GeminiMD != "" {
		parts = append(parts, ctx.GeminiMD)
	}
	if ctx.CursorRules != "" {
		parts = append(parts, ctx.CursorRules)
	}
	if userSystem != "" {
		parts = append(parts, userSystem)
	}

	return strings.Join(parts, "\n\n")
}

func readFileIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
