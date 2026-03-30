package repl

import "strings"

// InputKind identifies how user input should be dispatched.
type InputKind int

const (
	// InputPrompt is the default — natural language to GenSec.
	InputPrompt InputKind = iota

	// InputShell routes to the host shell (! prefix).
	InputShell

	// InputCommand routes to the internal command registry (: prefix).
	InputCommand

	// InputSlash routes to slash commands (/ prefix, legacy compat).
	InputSlash
)

// String returns a human-readable name for the input kind.
func (k InputKind) String() string {
	switch k {
	case InputShell:
		return "shell"
	case InputCommand:
		return "command"
	case InputSlash:
		return "slash"
	default:
		return "prompt"
	}
}

// ClassifyInput determines how to route user input based on its prefix.
// Returns the input kind and the payload with the prefix stripped.
//
//	"! ls -la"     → (InputShell, "ls -la")
//	":g E2"        → (InputCommand, "g E2")
//	"/help"        → (InputSlash, "help")
//	"fix the bug"  → (InputPrompt, "fix the bug")
//	""             → (InputPrompt, "")
func ClassifyInput(raw string) (kind InputKind, payload string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return InputPrompt, ""
	}

	switch trimmed[0] {
	case '!':
		return InputShell, strings.TrimSpace(trimmed[1:])
	case ':':
		return InputCommand, strings.TrimSpace(trimmed[1:])
	case '/':
		return InputSlash, strings.TrimSpace(trimmed[1:])
	default:
		return InputPrompt, trimmed
	}
}
