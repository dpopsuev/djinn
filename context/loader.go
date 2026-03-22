// Package context discovers and loads project instruction files
// from multiple CLI conventions (Claude, Codex, Gemini, Cursor, Aider).
// Walks up the directory tree to find files (like git finds .git).
// Reads MEMORY.md from Claude Code's project memory path.
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
	ClaudeMD    string   // CLAUDE.md or .claude/CLAUDE.md
	AgentsMD    string   // AGENTS.md (Codex)
	GeminiMD    string   // GEMINI.md
	CursorRules string   // .cursorrules or .cursor/rules
	AiderConf   string   // .aider.conf.yml
	MemoryMD    string   // ~/.claude/projects/<hash>/memory/MEMORY.md
	WorkDir     string   // primary workspace (backward compat)
	WorkDirs    []string // all workspace dirs
}

// LoadProjectContext discovers and reads project instruction files
// from the given directories. For each dir, walks up the tree to find
// instruction files (like git finds .git). Stops at $HOME.
// Falls back to single-dir behavior if called with a single string.
func LoadProjectContext(dirs ...string) ProjectContext {
	if len(dirs) == 0 {
		return ProjectContext{}
	}

	ctx := ProjectContext{
		WorkDir:  dirs[0],
		WorkDirs: dirs,
	}

	for _, dir := range dirs {
		// Walk up from each dir looking for instruction files
		walkUp(dir, func(d string) bool {
			if ctx.ClaudeMD == "" {
				ctx.ClaudeMD = readFileIfExists(filepath.Join(d, fileClaudeMD))
				if ctx.ClaudeMD == "" {
					ctx.ClaudeMD = readFileIfExists(filepath.Join(d, fileClaudeMDSub))
				}
			}
			if ctx.AgentsMD == "" {
				ctx.AgentsMD = readFileIfExists(filepath.Join(d, fileAgentsMD))
			}
			if ctx.GeminiMD == "" {
				ctx.GeminiMD = readFileIfExists(filepath.Join(d, fileGeminiMD))
			}
			if ctx.CursorRules == "" {
				ctx.CursorRules = readFileIfExists(filepath.Join(d, fileCursorRules))
				if ctx.CursorRules == "" {
					ctx.CursorRules = readFileIfExists(filepath.Join(d, fileCursorDir))
				}
			}
			if ctx.AiderConf == "" {
				ctx.AiderConf = readFileIfExists(filepath.Join(d, fileAiderConf))
			}
			// Stop if we found at least one file
			return ctx.ClaudeMD != "" && ctx.AgentsMD != "" && ctx.GeminiMD != ""
		})
	}

	// Discover MEMORY.md from Claude Code's project memory path
	ctx.MemoryMD = discoverMemory(dirs)

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
	if ctx.MemoryMD != "" {
		parts = append(parts, ctx.MemoryMD)
	}
	if userSystem != "" {
		parts = append(parts, userSystem)
	}

	return strings.Join(parts, "\n\n")
}

// walkUp traverses parent directories from dir up to $HOME or root.
// fn is called for each directory. If fn returns true, walk stops.
func walkUp(dir string, fn func(string) bool) {
	home, _ := os.UserHomeDir()
	dir, _ = filepath.Abs(dir)

	for {
		if fn(dir) {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return // reached filesystem root
		}
		if dir == home {
			return // don't walk above home
		}
		dir = parent
	}
}

// discoverMemory finds MEMORY.md from Claude Code's project memory path.
// Claude Code stores memory at: ~/.claude/projects/<slug>/memory/MEMORY.md
// where slug is the workspace path with slashes replaced by dashes.
func discoverMemory(dirs []string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	for _, dir := range dirs {
		absDir, _ := filepath.Abs(dir)
		// Claude Code slug: /home/user/Workspace/djinn → -home-user-Workspace-djinn
		slug := strings.ReplaceAll(absDir, "/", "-")
		memPath := filepath.Join(home, ".claude", "projects", slug, "memory", "MEMORY.md")
		if content := readFileIfExists(memPath); content != "" {
			return content
		}
	}
	return ""
}

func readFileIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
