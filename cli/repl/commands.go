package repl

import (
	"fmt"
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
	Output  string
	Exit    bool
	Cleared bool
}

// Slash command names.
const (
	cmdHelp   = "/help"
	cmdClear  = "/clear"
	cmdExit   = "/exit"
	cmdQuit   = "/quit"
	cmdModel  = "/model"
	cmdStatus = "/status"
	cmdCost   = "/cost"
)

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

	case cmdHelp:
		return CommandResult{Output: helpText()}

	default:
		return CommandResult{Output: fmt.Sprintf("unknown command: %s (try /help)", cmd.Name)}
	}
}

func helpText() string {
	return `commands:
  /model [name]  show or switch model
  /status        session info
  /cost          token usage
  /clear         clear conversation history
  /help          show this help
  /exit          quit`
}
