// turn.go — TurnPanel represents a single conversation turn.
// Layout: user input TOP, tool calls MIDDLE, agent output BELOW, thinking BOTTOM.
// Children() returns EnvelopePanels for tool calls — enables drill-down.
// No domain imports — pure TUI component.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TurnPanel wraps a conversation turn with drillable tool call children.
type TurnPanel struct {
	BasePanel
	userPrompt string  // what the user asked
	agentText  string  // agent response text
	thinking   string  // thinking/reasoning text
	toolCalls  []Panel // EnvelopePanels for each tool call
	tokenIn    int
	tokenOut   int
}

var _ Panel = (*TurnPanel)(nil)

// NewTurnPanel creates a turn panel.
func NewTurnPanel(id, userPrompt, agentText, thinking string, toolCalls []Panel, tokIn, tokOut int) *TurnPanel {
	return &TurnPanel{
		BasePanel:  NewBasePanel(id, 1),
		userPrompt: userPrompt,
		agentText:  agentText,
		thinking:   thinking,
		toolCalls:  toolCalls,
		tokenIn:    tokIn,
		tokenOut:   tokOut,
	}
}

// Children returns tool call panels — enables Dive into individual tool calls.
func (p *TurnPanel) Children() []Panel { return p.toolCalls }
func (p *TurnPanel) Collapsible() bool { return true }

func (p *TurnPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

// View renders the turn. Collapsed = one-line summary. Expanded = full turn.
func (p *TurnPanel) View(width int) string {
	if p.collapsed {
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
		UserStyle.Render(LabelUser),
		prompt,
		DimStyle.Render(meta))
}

func (p *TurnPanel) expandedView(width int) string {
	var sb strings.Builder

	// User prompt at TOP.
	sb.WriteString(UserStyle.Render(LabelUser) + p.userPrompt)

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
		sb.WriteString(DimStyle.Render(SpinnerFrames[0] + " " + p.thinking))
	}

	// Token stats.
	if p.tokenIn > 0 || p.tokenOut > 0 {
		sb.WriteByte('\n')
		sb.WriteString(StatusStyle.Render(fmt.Sprintf("[tokens: %d in, %d out]", p.tokenIn, p.tokenOut)))
	}

	return sb.String()
}
