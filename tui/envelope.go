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
	if msg, ok := msg.(ToolResultMsg); ok {
		p.SetResult(msg.Output, msg.IsError)
	}
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
	state := StateDone
	if p.isError {
		state = StateError
	}
	if !p.done {
		state = StateActive
	}
	return "  " + ToolStatus(p.toolName, state, lines)
}

func (p *EnvelopePanel) expandedView(width int) string {
	var sb strings.Builder
	state := StateActive
	if p.done {
		state = StateDone
	}
	if p.isError {
		state = StateError
	}
	fmt.Fprintf(&sb, "  %s %s\n",
		ToolStatus(p.toolName, state, 0),
		ToolArgStyle.Render(p.args))
	if p.output != "" {
		wrapped := WrapText(p.output, width-4)
		for _, line := range strings.Split(wrapped, "\n") {
			sb.WriteString("    " + DimStyle.Render(line) + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}
