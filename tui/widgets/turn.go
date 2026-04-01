// turn.go — TurnPanel represents a single conversation turn.
// Layout: user input TOP, tool calls MIDDLE, agent output BELOW, thinking BOTTOM.
// Children() returns EnvelopePanels for tool calls — enables drill-down.
// No domain imports — pure TUI component.
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// TurnPanel wraps a conversation turn with drillable tool call children.
type TurnPanel struct {
	core.BasePanel
	userPrompt string       // what the user asked
	agentText  string       // agent response text
	thinking   string       // thinking/reasoning text
	toolCalls  []core.Panel // EnvelopePanels for each tool call
	tokenIn    int
	tokenOut   int
}

var _ core.Panel = (*TurnPanel)(nil)

// NewTurnPanel creates a turn panel.
func NewTurnPanel(id, userPrompt, agentText, thinking string, toolCalls []core.Panel, tokIn, tokOut int) *TurnPanel {
	return &TurnPanel{
		BasePanel:  core.NewBasePanel(id, 1),
		userPrompt: userPrompt,
		agentText:  agentText,
		thinking:   thinking,
		toolCalls:  toolCalls,
		tokenIn:    tokIn,
		tokenOut:   tokOut,
	}
}

// Children returns tool call panels — enables Dive into individual tool calls.
func (p *TurnPanel) Children() []core.Panel { return p.toolCalls }
func (p *TurnPanel) Collapsible() bool      { return true }

func (p *TurnPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	return p, nil
}

// View renders the turn. Collapsed = one-line summary. Expanded = full turn.
func (p *TurnPanel) View(width int) string {
	if p.Collapsed() {
		return p.summaryView(width)
	}
	return p.expandedView(width)
}

func (p *TurnPanel) summaryView(width int) string {
	prompt := p.userPrompt
	maxLen := width - 30
	if maxLen > 0 && len(prompt) > maxLen {
		prompt = prompt[:maxLen-3] + "..."
	}
	meta := fmt.Sprintf("[%d tok, %d tools]", p.tokenIn+p.tokenOut, len(p.toolCalls))
	return fmt.Sprintf("%s %s %s",
		tui.UserStyle.Render(tui.LabelUser),
		prompt,
		tui.DimStyle.Render(meta))
}

func (p *TurnPanel) expandedView(width int) string {
	var sb strings.Builder

	// User prompt at TOP.
	sb.WriteString(tui.UserStyle.Render(tui.LabelUser) + p.userPrompt)

	// Tool calls in MIDDLE.
	for _, tc := range p.toolCalls {
		sb.WriteByte('\n')
		sb.WriteString(tc.View(width))
	}

	// Agent output BELOW.
	if p.agentText != "" {
		sb.WriteByte('\n')
		sb.WriteString(p.agentText)
	}

	// Thinking at BOTTOM.
	if p.thinking != "" {
		sb.WriteByte('\n')
		sb.WriteString(tui.DimStyle.Render(tui.SpinnerFrames[0] + " " + p.thinking))
	}

	// Token stats.
	if p.tokenIn > 0 || p.tokenOut > 0 {
		sb.WriteByte('\n')
		sb.WriteString(tui.StatusStyle.Render(fmt.Sprintf("[tokens: %d in, %d out]", p.tokenIn, p.tokenOut)))
	}

	return sb.String()
}
