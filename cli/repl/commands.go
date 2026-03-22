package repl

import (
	"fmt"
	"os/exec"
	"strings"

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
		return CommandResult{Output: "tools: Read, Write, Edit, Bash, Glob, Grep\napproval: ask (default)"}

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
		return executeConfig(sess)

	case cmdHelp:
		return CommandResult{Output: helpText()}

	default:
		return CommandResult{Output: fmt.Sprintf("unknown command: %s (try /help)", cmd.Name)}
	}
}

func executeCompact(sess *session.Session) CommandResult {
	before := sess.History.Len()
	if before <= 4 {
		return CommandResult{Output: "nothing to compact (history too short)"}
	}

	// Keep last 4 entries, summarize the rest
	entries := sess.Entries()
	keepFrom := len(entries) - 4
	old := entries[:keepFrom]
	recent := entries[keepFrom:]

	var summaryParts []string
	for _, e := range old {
		first := e.Content
		if idx := strings.IndexByte(first, '\n'); idx >= 0 {
			first = first[:idx]
		}
		const maxLine = 60
		if len(first) > maxLine {
			first = first[:maxLine] + "..."
		}
		if first != "" {
			summaryParts = append(summaryParts, e.Role+": "+first)
		}
	}

	sess.History.Clear()
	if len(summaryParts) > 0 {
		sess.Append(session.Entry{
			Role:    driver.RoleUser,
			Content: "[Compacted history]\n" + strings.Join(summaryParts, "\n"),
		})
	}
	for _, e := range recent {
		sess.Append(e)
	}

	after := sess.History.Len()
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
	return CommandResult{
		Output: fmt.Sprintf("session: %s\nmodel: %s\nworkdir: %s\nturns: %d\ntokens: ~%d",
			sess.ID, sess.Model, sess.WorkDir, sess.History.Len(), sess.TotalTokens()),
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

func executeConfig(sess *session.Session) CommandResult {
	mode := sess.Mode
	if mode == "" {
		mode = defaultModeName
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
  /clear           clear conversation history
  /help            show this help
  /exit            quit`
}
