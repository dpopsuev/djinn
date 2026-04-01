// agent_status.go — AgentStatusPanel shows a single agent's live status.
// Used as an item inside AgentsPanel (roster view).
package widgets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// Agent state constants.
const (
	AgentStateIdle      = "idle"
	AgentStateStreaming = "streaming"
	AgentStateToolWait  = "tool-wait"
	AgentStateDone      = "done"
	AgentStateError     = "error"
)

// AgentStatusPanel displays a single agent's role, state, and token usage.
type AgentStatusPanel struct {
	core.BasePanel
	AgentID   string
	Role      string
	Model     string
	State     string // "idle", "streaming", "tool-wait", "done", "error"
	TokensIn  int
	TokensOut int
	Color     lipgloss.Color
	output    *OutputPanel // per-agent output buffer for drill-down
}

// NewAgentStatusPanel creates a status card for one agent.
func NewAgentStatusPanel(agentID, role string, color lipgloss.Color) *AgentStatusPanel {
	return &AgentStatusPanel{
		BasePanel: core.NewBasePanel("agent-"+agentID, 1),
		AgentID:   agentID,
		Role:      role,
		State:     AgentStateIdle,
		Color:     color,
		output:    NewOutputPanel(),
	}
}

func (p *AgentStatusPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.AgentStatusMsg:
		if msg.AgentID == p.AgentID {
			p.State = msg.State
			p.Role = msg.Role
			p.TokensIn = msg.TokensIn
			p.TokensOut = msg.TokensOut
			if msg.Color != "" {
				p.Color = lipgloss.Color(msg.Color)
			}
		}
	case tui.AgentOutputMsg:
		if msg.AgentID == p.AgentID {
			p.output.Update(tui.OutputAppendMsg{Line: msg.Text})
		}
	}
	return p, nil
}

func (p *AgentStatusPanel) View(width int) string {
	stateStyle := tui.DimStyle
	switch p.State {
	case AgentStateStreaming:
		stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	case AgentStateToolWait:
		stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	case AgentStateError:
		stateStyle = tui.ErrorStyle
	}

	roleStyle := lipgloss.NewStyle().Foreground(p.Color).Bold(true)
	tokStr := ""
	if p.TokensIn > 0 || p.TokensOut > 0 {
		tokStr = fmt.Sprintf(" %d/%d tok", p.TokensIn, p.TokensOut)
	}

	return fmt.Sprintf("%s (%s%s)",
		roleStyle.Render(p.Role),
		stateStyle.Render(p.State),
		tui.DimStyle.Render(tokStr))
}

// Children returns the agent's output panel for Dive navigation.
func (p *AgentStatusPanel) Children() []core.Panel {
	return []core.Panel{p.output}
}

// Output returns the per-agent output panel.
func (p *AgentStatusPanel) Output() *OutputPanel {
	return p.output
}
