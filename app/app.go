// Package app contains all Djinn CLI domain logic.
// The cmd/djinn binary is a thin shell that calls this package.
// Per hexagonal-architecture rule: all domain logic lives in library modules.
package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/session"
)

// Version is set via -ldflags at build time. Falls back to "dev".
var Version = "0.1.0"

// Constants.
const (
	DefaultHomeDir    = ".djinn"
	DefaultSessionDir = ".djinn/sessions"
	DefaultModel      = "claude-sonnet-4-6"
	PollInterval      = 50 * time.Millisecond
)

// Driver names.
const (
	DriverClaude = "claude"
	DriverOllama = "ollama"
)

// HomeDir returns the Djinn home directory.
// Prefers XDG: ~/.config/djinn. Falls back to ~/.djinn for legacy installs.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "djinn")
	if _, err := os.Stat(configDir); err == nil {
		return configDir
	}
	djinnDir := filepath.Join(home, ".djinn")
	if _, err := os.Stat(djinnDir); err == nil {
		return djinnDir
	}
	return configDir
}

// SessionDir returns the default session directory.
func SessionDir() string {
	return filepath.Join(HomeDir(), "sessions")
}

// Run is the main entry point. Routes subcommands.
func Run(args []string, stderr io.Writer) error {
	if len(args) < 1 {
		return RunREPL(nil, stderr)
	}

	switch args[0] {
	case "repl":
		return RunREPL(args[1:], stderr)
	case "run":
		return RunHeadless(args[1:], stderr)
	case "ls":
		return RunList(stderr)
	case "attach":
		return RunAttach(args[1:], stderr)
	case "kill":
		return RunKill(args[1:], stderr)
	case "import":
		return RunImport(args[1:], stderr)
	case "config":
		return RunConfig(args[1:], stderr)
	case "workspace":
		return RunWorkspace(args[1:], stderr)
	case "log":
		return RunLog(stderr)
	case "doctor":
		return RunDoctor(stderr)
	case "debug":
		return RunDebug(args[1:], stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stderr, "djinn "+Version)
		return nil
	case "--help", "-h", "help":
		PrintUsage(stderr)
		return nil
	default:
		return RunREPL(args, stderr)
	}
}

// PrintUsage writes usage information.
func PrintUsage(w io.Writer) {
	fmt.Fprintf(w, `djinn — model-agnostic agent runtime

Usage:
  djinn [prompt]                      interactive REPL (default)
  djinn repl [flags] [prompt]         interactive REPL
  djinn run <prompt> [flags]          headless one-shot
  djinn import claude <file> -s <name> import Claude Code session
  djinn ls                            list sessions
  djinn attach <name>                 resume session
  djinn kill <name>                   delete session
  djinn workspace list                list saved workspaces
  djinn workspace create <name>       create a workspace
  djinn config dump                   dump runtime config as YAML
  djinn log                           show recent log entries
  djinn doctor                        health check
  djinn version                       version info

Flags (repl/run):
  --driver <claude|ollama>            LLM backend (default: claude)
  -m, --model <model>                 model name
  -s, --session <name>                named session
  -c, --continue                      resume most recent session
  --max-turns <n>                     max agent turns (default: 20)
  --auto-approve                      auto-approve all tool calls
  --system <prompt>                   system prompt
  --system-file <path>                load system prompt from file
  --mode <ask|plan|agent|auto>        agent mode (default: agent)
  --config <file>                     load config from YAML file
  -w, --workspace <name|file>          workspace name or manifest file
  --verbose                           show log output on terminal
  --no-persist                        don't save session to disk
`)
}

// LoadMostRecent loads the most recently updated session.
func LoadMostRecent(store *session.Store) (*session.Session, error) {
	list, err := store.List()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNoSessions
	}
	name := list[0].Name
	if name == "" {
		name = list[0].ID
	}
	return store.Load(name)
}

// ReadSystemFile reads a system prompt from a file. Returns empty on error.
func ReadSystemFile(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Getwd returns the current working directory.
func Getwd() string {
	d, _ := os.Getwd()
	return d
}
