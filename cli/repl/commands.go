package repl

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
)

// Command represents a parsed slash command.
type Command struct {
	Name string
	Args []string
}

// CommandResult is the outcome of executing a slash command.
type CommandResult struct {
	Output     string
	Exit       bool
	Cleared    bool
	ModeChange string // non-empty = mode was changed
}

// Slash command names.
const (
	cmdHelp        = "/help"
	cmdClear       = "/clear"
	cmdExit        = "/exit"
	cmdQuit        = "/quit"
	cmdModel       = "/model"
	cmdStatus      = "/status"
	cmdCost        = "/cost"
	cmdCompact     = "/compact"
	cmdDiff        = "/diff"
	cmdMode        = "/mode"
	cmdPermissions = "/permissions"
	cmdMemory      = "/memory"
	cmdMcp         = "/mcp"
	cmdResume      = "/resume"
	cmdCopy        = "/copy"
	cmdReview      = "/review"
	cmdOutput      = "/output"
	cmdConfig      = "/config"
	cmdLog         = "/log"
	cmdWorkspace   = "/workspace"
)

// Default mode name.
const defaultModeName = "agent"

// ParseCommand parses a slash command string into a Command.
func ParseCommand(input string) (Command, bool) {
	if !strings.HasPrefix(input, "/") {
		return Command{}, false
	}
	parts := strings.Fields(input)
	return Command{
		Name: parts[0],
		Args: parts[1:],
	}, true
}

// ExecuteCommand runs a slash command and returns the result.
func ExecuteCommand(cmd Command, sess *session.Session) CommandResult {
	switch cmd.Name {
	case cmdExit, cmdQuit:
		return CommandResult{Output: "goodbye.", Exit: true}

	case cmdClear:
		sess.History.Clear()
		return CommandResult{Output: "history cleared.", Cleared: true}

	case cmdModel:
		if len(cmd.Args) > 0 {
			sess.Model = cmd.Args[0]
			return CommandResult{Output: fmt.Sprintf("model set to %s", cmd.Args[0])}
		}
		return CommandResult{Output: fmt.Sprintf("current model: %s", sess.Model)}

	case cmdStatus:
		return CommandResult{
			Output: fmt.Sprintf("session: %s | model: %s | turns: %d | tokens: ~%d",
				sess.ID, sess.Model, sess.History.Len(), sess.TotalTokens()),
		}

	case cmdCost:
		return CommandResult{
			Output: fmt.Sprintf("tokens used: ~%d (approximate)", sess.TotalTokens()),
		}

	case cmdCompact:
		return executeCompact(sess)

	case cmdDiff:
		return executeDiff()

	case cmdMode:
		return executeMode(cmd, sess)

	case cmdPermissions:
		return executePermissions(sess)

	case cmdMemory:
		return executeMemory(sess)

	case cmdMcp:
		return CommandResult{Output: "mcp servers: (use --mcp-config to configure)"}

	case cmdResume:
		return CommandResult{Output: "use 'djinn ls' to list sessions, 'djinn attach <name>' to resume"}

	case cmdCopy:
		return executeCopy(sess)

	case cmdReview:
		return CommandResult{Output: "review: (send /review as a prompt to request agent code review)"}

	case cmdOutput:
		return executeOutput(cmd)

	case cmdConfig:
		return executeConfig(cmd, sess)

	case cmdLog:
		return executeLog(cmd)

	case cmdWorkspace:
		return executeWorkspace(cmd, sess)

	case cmdHelp:
		return CommandResult{Output: helpText()}

	default:
		return CommandResult{Output: fmt.Sprintf("unknown command: %s (try /help)", cmd.Name)}
	}
}

func executeCompact(sess *session.Session) CommandResult {
	before, after := session.Compact(sess, session.DefaultKeepRecent)
	if before == after {
		return CommandResult{Output: "nothing to compact (history too short)"}
	}
	return CommandResult{
		Output: fmt.Sprintf("compacted: %d → %d turns", before, after),
	}
}

func executeDiff() CommandResult {
	out, err := exec.Command("git", "diff", "--stat").Output()
	if err != nil {
		return CommandResult{Output: "no git diff available"}
	}
	diff := strings.TrimSpace(string(out))
	if diff == "" {
		return CommandResult{Output: "no changes (working tree clean)"}
	}
	return CommandResult{Output: diff}
}

func executeMode(cmd Command, sess *session.Session) CommandResult {
	currentMode := sess.Mode
	if currentMode == "" {
		currentMode = defaultModeName
	}

	if len(cmd.Args) > 0 {
		newMode := cmd.Args[0]
		switch newMode {
		case "ask", "plan", "agent", "auto":
			sess.Mode = newMode
			return CommandResult{
				Output:     fmt.Sprintf("mode: %s", newMode),
				ModeChange: newMode,
			}
		default:
			return CommandResult{Output: fmt.Sprintf("unknown mode: %s (available: ask, plan, agent, auto)", newMode)}
		}
	}
	return CommandResult{Output: fmt.Sprintf("current mode: %s", currentMode)}
}

func executeOutput(cmd Command) CommandResult {
	if len(cmd.Args) == 0 {
		return CommandResult{Output: "output modes:\n  /output streaming  — token-by-token (default)\n  /output chunked    — all-at-once after completion\n  /output follow     — auto-scroll to latest (default)\n  /output static     — manual scroll"}
	}
	switch cmd.Args[0] {
	case "streaming", "chunked", "follow", "static":
		return CommandResult{Output: fmt.Sprintf("output mode set to: %s", cmd.Args[0])}
	default:
		return CommandResult{Output: fmt.Sprintf("unknown output mode: %s (streaming, chunked, follow, static)", cmd.Args[0])}
	}
}

func executeMemory(sess *session.Session) CommandResult {
	name := sess.Name
	if name == "" {
		name = sess.ID
	}
	return CommandResult{
		Output: fmt.Sprintf("Session:   %s\nModel:     %s\nWorkdir:   %s\nTurns:     %d\nTokens:    ~%d",
			name, sess.Model, sess.WorkDir, sess.History.Len(), sess.TotalTokens()),
	}
}

func executeCopy(sess *session.Session) CommandResult {
	entries := sess.Entries()
	if len(entries) == 0 {
		return CommandResult{Output: "nothing to copy (empty history)"}
	}

	// Find last assistant message
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Role == driver.RoleAssistant {
			return CommandResult{Output: "copied last response to clipboard (stub — clipboard integration pending)"}
		}
	}
	return CommandResult{Output: "no assistant response to copy"}
}

func executePermissions(sess *session.Session) CommandResult {
	mode := sess.Mode
	if mode == "" {
		mode = defaultModeName
	}
	approval := "interactive"
	switch mode {
	case "auto":
		approval = "auto-approve (no prompts)"
	case "ask", "plan":
		approval = "none (tools disabled)"
	}
	return CommandResult{
		Output: fmt.Sprintf("tools: Read, Write, Edit, Bash, Glob, Grep\napproval: %s\nmode: %s", approval, mode),
	}
}

func executeWorkspace(cmd Command, sess *session.Session) CommandResult {
	if len(cmd.Args) == 0 {
		if len(sess.WorkDirs) == 0 {
			return CommandResult{Output: fmt.Sprintf("workspace: %s", sess.WorkDir)}
		}
		var sb strings.Builder
		for i, d := range sess.WorkDirs {
			if i == 0 {
				fmt.Fprintf(&sb, "  %s (primary)\n", d)
			} else {
				fmt.Fprintf(&sb, "  %s\n", d)
			}
		}
		return CommandResult{Output: strings.TrimRight(sb.String(), "\n")}
	}
	if cmd.Args[0] == "add" && len(cmd.Args) > 1 {
		dir := cmd.Args[1]
		sess.WorkDirs = append(sess.WorkDirs, dir)
		return CommandResult{Output: fmt.Sprintf("added workspace: %s (%d dirs total)", dir, len(sess.WorkDirs))}
	}
	return CommandResult{Output: "usage: /workspace [add <path>]"}
}

func executeLog(cmd Command) CommandResult {
	if globalRing == nil {
		return CommandResult{Output: "logging not initialized"}
	}

	// Parse args: /log [count|level]
	count := 20
	var levelFilter *slog.Level
	if len(cmd.Args) > 0 {
		switch cmd.Args[0] {
		case "error":
			lvl := slog.LevelError
			levelFilter = &lvl
		case "warn":
			lvl := slog.LevelWarn
			levelFilter = &lvl
		case "info":
			lvl := slog.LevelInfo
			levelFilter = &lvl
		case "debug":
			lvl := slog.LevelDebug
			levelFilter = &lvl
		default:
			if n, err := strconv.Atoi(cmd.Args[0]); err == nil && n > 0 {
				count = n
			}
		}
	}

	var entries []djinnlog.Entry
	if levelFilter != nil {
		entries = globalRing.Filter(*levelFilter)
	} else {
		entries = globalRing.Entries()
	}

	// Take last N
	if len(entries) > count {
		entries = entries[len(entries)-count:]
	}

	if len(entries) == 0 {
		return CommandResult{Output: "no log entries"}
	}

	var sb strings.Builder
	for _, e := range entries {
		comp := e.Component
		if comp == "" {
			comp = "-"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s %-8s %s\n",
			e.Time.Format("15:04:05"), e.Level.String(), comp, e.Message))
	}
	return CommandResult{Output: strings.TrimRight(sb.String(), "\n")}
}

func executeConfig(cmd Command, sess *session.Session) CommandResult {
	mode := sess.Mode
	if mode == "" {
		mode = defaultModeName
	}

	if len(cmd.Args) > 0 && cmd.Args[0] == "save" {
		path := "djinn.yaml"
		if len(cmd.Args) > 1 {
			path = cmd.Args[1]
		}
		content := fmt.Sprintf("driver:\n  name: %s\n  model: %s\nmode: %s\nsession:\n  max_turns: 20\n",
			sess.Driver, sess.Model, mode)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return CommandResult{Output: fmt.Sprintf("error saving config: %v", err)}
		}
		return CommandResult{Output: fmt.Sprintf("config saved to %s", path)}
	}

	return CommandResult{
		Output: fmt.Sprintf("driver: %s\nmodel: %s\nmode: %s\nturns: %d\nworkdir: %s",
			sess.Driver, sess.Model, mode, sess.History.Len(), sess.WorkDir),
	}
}

func helpText() string {
	return `commands:
  /model [name]    show or switch model
  /mode [mode]     show or switch mode (ask, plan, agent, auto)
  /output [mode]   output mode (streaming, chunked, follow, static)
  /status          session info
  /cost            token usage
  /compact         compress conversation history
  /diff            show git diff
  /copy            copy last response
  /permissions     show tool access
  /memory          show session details
  /mcp             show MCP servers
  /resume          resume a session (use djinn attach)
  /review          request code review
  /config          show runtime config
  /log [n|level]   show recent log entries
  /workspace [add] show or add workspace directories
  /clear           clear conversation history
  /help            show this help
  /exit            quit`
}
