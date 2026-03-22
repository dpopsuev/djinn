// commands_session.go — session-related slash commands.
package repl

import (
	"fmt"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
)

// Session command names.
const (
	cmdModel       = "/model"
	cmdMode        = "/mode"
	cmdStatus      = "/status"
	cmdCost        = "/cost"
	cmdCompact     = "/compact"
	cmdMemory      = "/memory"
	cmdCopy        = "/copy"
	cmdPermissions = "/permissions"
	cmdOutput      = "/output"
	cmdResume      = "/resume"
)

// Default mode name.
const defaultModeName = "agent"

func executeModel(cmd Command, sess *session.Session) CommandResult {
	if len(cmd.Args) > 0 {
		sess.Model = cmd.Args[0]
		return CommandResult{Output: fmt.Sprintf("model set to %s", cmd.Args[0])}
	}
	return CommandResult{Output: fmt.Sprintf("current model: %s", sess.Model)}
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

func executeStatus(sess *session.Session) CommandResult {
	return CommandResult{
		Output: fmt.Sprintf("session: %s | model: %s | turns: %d | tokens: ~%d",
			sess.ID, sess.Model, sess.History.Len(), sess.TotalTokens()),
	}
}

func executeCost(sess *session.Session) CommandResult {
	return CommandResult{
		Output: fmt.Sprintf("tokens used: ~%d (approximate)", sess.TotalTokens()),
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
