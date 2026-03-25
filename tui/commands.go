// commands.go — CommandsPanel shows matching slash commands.
// Visible only when input starts with /. Selectable list.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandsPanel displays slash command suggestions.
type CommandsPanel struct {
	BasePanel
	commands []string // all available commands
	matches  []string // filtered matches
	visible  bool
	cursor   int
}

const panelIDCommands = "commands"

var _ Panel = (*CommandsPanel)(nil)

func NewCommandsPanel(commands []string) *CommandsPanel {
	return &CommandsPanel{
		BasePanel: NewBasePanel(panelIDCommands, 0),
		commands:  commands,
	}
}

// Active returns whether the panel has content to show.
func (p *CommandsPanel) Active() bool { return p.visible && len(p.matches) > 0 }

// Selected returns the currently highlighted command.
func (p *CommandsPanel) Selected() string {
	if p.cursor >= 0 && p.cursor < len(p.matches) {
		return p.matches[p.cursor]
	}
	return ""
}

func (p *CommandsPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case CommandsShowMsg:
		p.visible = true
		p.matches = filterCommands(p.commands, msg.Filter)
		p.cursor = 0
	case CommandsHideMsg:
		p.visible = false
		p.matches = nil
		p.cursor = 0
	case tea.KeyMsg:
		if !p.visible || !p.focused {
			return p, nil
		}
		switch msg.Type {
		case tea.KeyUp:
			if p.cursor > 0 {
				p.cursor--
			}
		case tea.KeyDown:
			if p.cursor < len(p.matches)-1 {
				p.cursor++
			}
		}
	}
	return p, nil
}

func (p *CommandsPanel) View(width int) string {
	if !p.visible || len(p.matches) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, cmd := range p.matches {
		prefix := "  "
		if i == p.cursor {
			prefix = DimStyle.Render("▸ ")
		}
		sb.WriteString(prefix + DimStyle.Render(cmd))
		if i < len(p.matches)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func filterCommands(commands []string, prefix string) []string {
	var out []string
	for _, c := range commands {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}
