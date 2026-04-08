// commands_model.go — slash commands that require Model access.
//
// Extracted from handleSubmit (SRP: separate command handling from input routing).
// These commands read/mutate Model state (roles, sandbox, debug panels).
// No bubbletea import — depguard restricts it to tui/ only.
package repl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/tui/elements"
	"github.com/dpopsuev/djinn/uniform"
)

// handleRoleCmd handles /role — list, create, or switch roles.
func (m *Model) handleRoleCmd(cmd Command) {
	switch {
	case len(cmd.Args) == 0:
		names := make([]string, 0, len(m.roles))
		for n := range m.roles {
			names = append(names, n)
		}
		sort.Strings(names)
		m.outputPanel.Update(tui.OutputAppendMsg{Line: fmt.Sprintf("current role: %s\navailable: %s",
			m.currentRole, strings.Join(names, ", "))})
	case cmd.Args[0] == "create" && len(cmd.Args) >= 3:
		name, mode := cmd.Args[1], cmd.Args[2]
		m.roles[name] = uniform.Role{
			Name:             name,
			Prompt:           fmt.Sprintf("You are %s. The operator created this role on the fly.", name),
			Mode:             mode,
			ToolCapabilities: []string{},
		}
		m.outputPanel.Update(tui.OutputAppendMsg{Line: fmt.Sprintf("created role %q (mode: %s, capabilities: none — use /role capabilities %s to configure)", name, mode, name)})
	default:
		m.switchRole(cmd.Args[0])
		m.outputPanel.Update(tui.OutputAppendMsg{Line: fmt.Sprintf("switched to %s (manual override)", cmd.Args[0])})
	}
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
}

// handleStaffCmd handles /staff — show all roles and their current state.
func (m *Model) handleStaffCmd() {
	var sb strings.Builder
	sb.WriteString("Staff:\n")
	names := make([]string, 0, len(m.roles))
	for n := range m.roles {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		role := m.roles[name]
		indicator := "  "
		if name == m.currentRole {
			indicator = "→ "
		}
		sb.WriteString(fmt.Sprintf("%s%s (mode: %s, slots: %d)\n",
			indicator, name, role.Mode, len(role.ToolCapabilities)))
	}
	m.outputPanel.Update(tui.OutputAppendMsg{Line: sb.String()})
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
}

// handleJailbreakCmd handles /jailbreak — toggle sandbox for GenSec only.
func (m *Model) handleJailbreakCmd() {
	switch {
	case m.currentRole != roleGensec:
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render("jailbreak: only GenSec (root agent) can toggle sandbox")})
	case m.sandboxHandle == "":
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("jailbreak: no sandbox configured")})
	default:
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("  GenSec is the root agent — always unsandboxed. Other roles are sandboxed by default.")})
	}
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
}

// handleBriefingCmd handles /briefing — show role memory briefing.
func (m *Model) handleBriefingCmd() {
	entries := m.roleMemory.Briefing()
	if len(entries) == 0 {
		m.outputPanel.Update(tui.OutputAppendMsg{Line: "briefing: (empty)"})
	} else {
		var sb strings.Builder
		sb.WriteString("Briefing:\n")
		for _, e := range entries {
			ts := e.Timestamp.Format("15:04:05")
			fmt.Fprintf(&sb, "  [%s] %s\n", ts, e.Content)
		}
		m.outputPanel.Update(tui.OutputAppendMsg{Line: sb.String()})
	}
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
}

// handleExecCmd handles /exec — run a shell command inline.
// Returns the shell command string to execute, or empty if usage error.
func (m *Model) handleExecCmd(cmd Command) string {
	if len(cmd.Args) == 0 {
		m.outputPanel.Update(tui.OutputAppendMsg{Line: "usage: /exec <command>"})
		m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
		return ""
	}
	return strings.Join(cmd.Args, " ")
}

// handleDebugCmd handles /debug — toggle trace debug panel.
func (m *Model) handleDebugCmd() {
	m.showDebug = !m.showDebug
	state := "off"
	if m.showDebug {
		state = "on"
	}
	m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("debug panel: " + state)})
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
}

// handleMCPCmd handles /mcp — list registered tools by source.
func (m *Model) handleMCPCmd() {
	var sb strings.Builder
	sb.WriteString("Registered Tools:\n")
	for _, name := range m.tools.Names() {
		fmt.Fprintf(&sb, "  %s %s\n", elements.Glyph(elements.StateDone), name)
	}
	sb.WriteString(elements.Dim(fmt.Sprintf("\n  %d tools total", len(m.tools.Names()))))
	m.outputPanel.Update(tui.OutputAppendMsg{Line: sb.String()})
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
}
