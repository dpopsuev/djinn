// commands_tools.go — development/tooling slash commands.
package repl

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tui"
)

// Tools & dev command names.
const (
	cmdDiff       = "/diff"
	cmdLog        = "/log"
	cmdConfig     = "/config"
	cmdConfigSave = "/config-save"
	cmdMcp        = "/mcp"
	cmdReview     = "/review"
)

func executeDiff() CommandResult {
	out, err := exec.Command("git", "diff").Output()
	if err != nil {
		return CommandResult{Output: "no git diff available"}
	}
	diff := strings.TrimSpace(string(out))
	if diff == "" {
		return CommandResult{Output: "no changes (working tree clean)"}
	}
	return CommandResult{Output: tui.RenderDiff(diff)}
}

func executeLog(cmd Command) CommandResult {
	if globalRing == nil {
		return CommandResult{Output: "logging not initialized"}
	}

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
		fmt.Fprintf(&sb, "[%s] %s %-8s %s\n",
			e.Time.Format("15:04:05"), e.Level.String(), comp, e.Message)
	}
	return CommandResult{Output: strings.TrimRight(sb.String(), "\n")}
}

func executeConfig(cmd Command, sess *session.Session) CommandResult {
	mode := sess.Mode
	if mode == "" {
		mode = defaultModeName
	}

	if len(cmd.Args) > 0 && cmd.Args[0] == "save" {
		return executeConfigSave(Command{Args: cmd.Args[1:]}, sess)
	}

	return CommandResult{
		Output: fmt.Sprintf("driver: %s\nmodel: %s\nmode: %s\nturns: %d\nworkdir: %s",
			sess.Driver, sess.Model, mode, sess.History.Len(), sess.WorkDir),
	}
}

func executeConfigSave(cmd Command, sess *session.Session) CommandResult {
	mode := sess.Mode
	if mode == "" {
		mode = defaultModeName
	}
	path := "djinn.yaml"
	if len(cmd.Args) > 0 {
		path = cmd.Args[0]
	}
	content := fmt.Sprintf("driver:\n  name: %s\n  model: %s\nmode: %s\nsession:\n  max_turns: 20\n",
		sess.Driver, sess.Model, mode)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return CommandResult{Output: fmt.Sprintf("error saving config: %v", err)}
	}
	return CommandResult{Output: fmt.Sprintf("config saved to %s", path)}
}
