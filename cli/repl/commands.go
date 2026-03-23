// commands.go — types, parser, dispatcher, help text.
package repl

import (
	"fmt"
	"sort"
	"strings"

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
	cmdHelp     = "/help"
	cmdClear    = "/clear"
	cmdExit     = "/exit"
	cmdQuit     = "/quit"
	cmdSessions = "/sessions"
)

// CommandNames returns all slash command names in sorted order.
func CommandNames() []string {
	names := []string{
		cmdHelp, cmdClear, cmdExit, cmdQuit, cmdSessions,
		cmdModel, cmdMode, cmdStatus, cmdCost, cmdCompact,
		cmdMemory, cmdCopy, cmdPermissions, cmdOutput, cmdResume,
		cmdWorkspace, cmdWorkspaceSwitch, cmdWorkspaceAdd,
		cmdWorkspaceRepos, cmdWorkspaceSave,
		cmdDiff, cmdLog, cmdConfig, cmdConfigSave, cmdMcp, cmdReview,
	}
	sort.Strings(names)
	return names
}

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
	// Lifecycle
	case cmdExit, cmdQuit:
		return CommandResult{Output: "goodbye.", Exit: true}
	case cmdClear:
		sess.History.Clear()
		return CommandResult{Output: "history cleared.", Cleared: true}
	case cmdHelp:
		return CommandResult{Output: helpText()}

	// Session
	case cmdModel:
		return executeModel(cmd, sess)
	case cmdMode:
		return executeMode(cmd, sess)
	case cmdStatus:
		return executeStatus(sess)
	case cmdCost:
		return executeCost(sess)
	case cmdCompact:
		return executeCompact(sess)
	case cmdMemory:
		return executeMemory(sess)
	case cmdCopy:
		return executeCopy(sess)
	case cmdPermissions:
		return executePermissions(sess)
	case cmdOutput:
		return executeOutput(cmd)
	case cmdResume:
		return CommandResult{Output: "use 'djinn ls' to list sessions, 'djinn attach <name>' to resume"}
	case cmdSessions:
		return executeSessions(cmd, sess)

	// Workspace
	case cmdWorkspace:
		return executeWorkspace(cmd, sess)
	case cmdWorkspaceSwitch:
		return executeWorkspaceSwitch(cmd, sess)
	case cmdWorkspaceAdd:
		return executeWorkspaceAdd(cmd, sess)
	case cmdWorkspaceRepos:
		return executeWorkspaceRepos(sess)
	case cmdWorkspaceSave:
		return executeWorkspaceSave(sess)

	// Tools & Dev
	case cmdDiff:
		return executeDiff()
	case cmdLog:
		return executeLog(cmd)
	case cmdConfig:
		return executeConfig(cmd, sess)
	case cmdConfigSave:
		return executeConfigSave(cmd, sess)
	case cmdMcp:
		return CommandResult{Output: "mcp servers: (use --mcp-config to configure)"}
	case cmdReview:
		return CommandResult{Output: "review: (send /review as a prompt to request agent code review)"}

	default:
		return CommandResult{Output: fmt.Sprintf("unknown command: %s (try /help)", cmd.Name)}
	}
}

func helpText() string {
	return `commands:
  /model [name]           show or switch model
  /mode [mode]            show or switch mode (ask, plan, agent, auto)
  /status                 session info
  /cost                   token usage
  /compact                compress conversation history
  /memory                 show session details
  /copy                   copy last response
  /permissions            show tool access
  /sessions [query]       search sessions (telescope)
  /output [mode]          output mode (streaming, chunked, follow, static)

  /workspace              show current workspace
  /workspace-switch <ws>  switch workspace (hot-swap)
  /workspace-add <path>   add repo to workspace
  /workspace-repos        list repos
  /workspace-save         persist workspace

  /diff                   show git diff
  /log [n|level]          show recent log entries
  /config                 show runtime config
  /config-save [file]     save config to YAML
  /mcp                    show MCP servers
  /review                 request code review

  /clear                  clear conversation history
  /help                   show this help
  /exit                   quit`
}
