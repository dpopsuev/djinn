// envelope.go — EnvelopePanel wraps a tool call + result (collapsible).
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// EnvelopePanel is a collapsible tool call + result.
type EnvelopePanel struct {
	BasePanel
	toolName string
	args     string
	output   string
	isError  bool
	done     bool // result received
}

var _ Panel = (*EnvelopePanel)(nil)

// NewEnvelopePanel creates an envelope for a tool call.
func NewEnvelopePanel(id, toolName, args string) *EnvelopePanel {
	return &EnvelopePanel{
		BasePanel: NewBasePanel(id, 0),
		toolName:  toolName,
		args:      args,
	}
}

func (p *EnvelopePanel) Collapsible() bool { return true }

// SetResult adds the tool result to the envelope.
func (p *EnvelopePanel) SetResult(output string, isError bool) {
	p.output = output
	p.isError = isError
	p.done = true
	p.collapsed = true // auto-collapse on result
}

func (p *EnvelopePanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *EnvelopePanel) View(width int) string {
	if p.collapsed {
		return p.summaryView()
	}
	return p.expandedView(width)
}

func (p *EnvelopePanel) summaryView() string {
	lines := strings.Count(p.output, "\n") + 1
	if p.isError {
		return fmt.Sprintf("  %s %s",
			ErrorStyle.Render("✗ "+p.toolName),
			DimStyle.Render(fmt.Sprintf("(error, %d lines)", lines)))
	}
	return fmt.Sprintf("  %s %s",
		ToolSuccessStyle.Render("✓ "+p.toolName),
		DimStyle.Render(fmt.Sprintf("(%d lines)", lines)))
}

func (p *EnvelopePanel) expandedView(width int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s %s\n",
		ToolNameStyle.Render(p.toolName),
		ToolArgStyle.Render(p.args)))
	if p.output != "" {
		wrapped := WrapText(p.output, width-4)
		for _, line := range strings.Split(wrapped, "\n") {
			sb.WriteString("    " + DimStyle.Render(line) + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}
