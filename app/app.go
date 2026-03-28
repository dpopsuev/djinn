// Package app contains all Djinn CLI domain logic.
// The cmd/djinn binary is a thin shell that calls this package.
// Per hexagonal-architecture rule: all domain logic lives in library modules.
package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/session"
)

// Version is set via -ldflags at build time.
// Falls back to git hash from debug.ReadBuildInfo (Go 1.18+).
var Version = "0.1.0"

func init() {
	if Version != "0.1.0" {
		return // ldflags already set
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	var revision, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision != "" {
		if len(revision) > 8 {
			revision = revision[:8]
		}
		Version = revision + dirty
	}
}

// Constants.
const (
	DefaultHomeDir    = ".djinn"
	DefaultSessionDir = ".djinn/sessions"
	DefaultModel      = "" // no default — djinn.yaml must specify
	PollInterval      = 50 * time.Millisecond
)

// Driver names.
const (
	DriverClaude    = "claude"
	DriverClaudeAPI = "claude-api"
	DriverOllama    = "ollama"
	DriverCursor    = "cursor"
	DriverGemini    = "gemini"
	DriverCodex     = "codex"
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
	case "hub":
		return RunHub(args[1:], stderr)
	case "backend":
		return RunBackendCmd(args[1:], stderr)
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
  djinn run <prompt> [flags]          headless one-shot
  djinn ls                            list sessions
  djinn attach <name>                 resume session
  djinn kill <name>                   delete session
  djinn import claude <file> -s <name> import Claude Code session
  djinn config dump                   dump runtime config as YAML
  djinn doctor                        health check
  djinn hub                           start GenSec daemon (hot-swap relay)
  djinn backend --socket <path>       headless backend (connects to hub)
  djinn version                       version info

Essential flags:
  -m <model>                          model name
  -s <name>                           named session
  -c                                  resume most recent session
  -e <ecosystem>                      scope (e.g. aeon, aeon/djinn)
  --config <file>                     config file (default: djinn.yaml)
  --socket <path>                     Unix socket for hot-swap

Override flags (prefer djinn.yaml):
  --driver <claude|ollama>            LLM backend
  --mode <ask|plan|agent|auto>        agent mode
  --system <prompt>                   system prompt
  --system-file <path>                system prompt from file
  --verbose                           show log output

Config (djinn.yaml):
  driver.name, driver.model, mode, session.max_turns,
  session.auto_approve, session.no_persist, sandbox.backend,
  sandbox.level, debug.tap_file, debug.live_debug, debug.verbose
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
