// agents.go — AgentsPanel is a roster of running agents with drill-down.
// Visible when multiple agents are running. Cursor selects an agent.
// Enter dives into the selected agent's output. Esc climbs back.
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// Cursor indicators for the agent roster.
const (
	cursorActive   = "> "
	cursorInactive = "  "
)

// AgentsPanel manages a roster of agent status cards.
type AgentsPanel struct {
	core.BasePanel
	agents []*AgentStatusPanel
	cursor int
}

// NewAgentsPanel creates an empty agents roster.
func NewAgentsPanel() *AgentsPanel {
	return &AgentsPanel{
		BasePanel: core.NewBasePanel("agents", 5),
	}
}

// Count returns the number of registered agents.
func (p *AgentsPanel) Count() int {
	return len(p.agents)
}

// AddAgent registers a new agent in the roster.
func (p *AgentsPanel) AddAgent(agentID, role string, color lipgloss.Color) {
	p.agents = append(p.agents, NewAgentStatusPanel(agentID, role, color))
}

// RemoveAgent removes an agent from the roster.
func (p *AgentsPanel) RemoveAgent(agentID string) {
	for i, a := range p.agents {
		if a.AgentID == agentID {
			p.agents = append(p.agents[:i], p.agents[i+1:]...)
			if p.cursor >= len(p.agents) && p.cursor > 0 {
				p.cursor--
			}
			return
		}
	}
}

// UpdateAgent updates an agent's status. If the agent is not yet registered,
// it is auto-added using the Color from the message (or a default).
func (p *AgentsPanel) UpdateAgent(msg tui.AgentStatusMsg) {
	for _, a := range p.agents {
		if a.AgentID == msg.AgentID {
			a.Update(msg)
			return
		}
	}
	// Auto-register on first status message.
	color := lipgloss.Color("7") // default: white
	if msg.Color != "" {
		color = lipgloss.Color(msg.Color)
	}
	p.AddAgent(msg.AgentID, msg.Role, color)
	p.UpdateAgent(msg) // apply the status update
}

// GetAgent returns the agent panel for the given ID.
func (p *AgentsPanel) GetAgent(agentID string) *AgentStatusPanel {
	for _, a := range p.agents {
		if a.AgentID == agentID {
			return a
		}
	}
	return nil
}

// Selected returns the currently selected agent, or nil.
func (p *AgentsPanel) Selected() *AgentStatusPanel {
	if p.cursor >= 0 && p.cursor < len(p.agents) {
		return p.agents[p.cursor]
	}
	return nil
}

func (p *AgentsPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.AgentStatusMsg:
		p.UpdateAgent(msg)
	case tui.AgentOutputMsg:
		for _, a := range p.agents {
			if a.AgentID == msg.AgentID {
				a.Update(msg)
			}
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if p.cursor > 0 {
				p.cursor--
			}
		case tea.KeyDown:
			if p.cursor < len(p.agents)-1 {
				p.cursor++
			}
		}
	}
	return p, nil
}

func (p *AgentsPanel) View(width int) string {
	if len(p.agents) == 0 {
		return tui.DimStyle.Render("no agents")
	}

	var sb strings.Builder
	for i, a := range p.agents {
		indicator := cursorInactive
		if i == p.cursor {
			indicator = cursorActive
		}
		line := fmt.Sprintf("%s%s", indicator, a.View(width-4))
		if i < len(p.agents)-1 {
			line += "\n"
		}
		sb.WriteString(line)
	}
	return sb.String()
}

// Children returns the selected agent's output panel for Dive.
func (p *AgentsPanel) Children() []core.Panel {
	if sel := p.Selected(); sel != nil {
		return sel.Children()
	}
	return nil
}
