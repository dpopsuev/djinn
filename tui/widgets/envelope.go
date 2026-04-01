// envelope.go — EnvelopePanel wraps a tool call + result (collapsible).
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	tui "github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/tui/core"
	"github.com/dpopsuev/djinn/tui/elements"
)

// EnvelopePanel is a collapsible tool call + result.
type EnvelopePanel struct {
	core.BasePanel
	toolName string
	args     string
	output   string
	isError  bool
	done     bool // result received
}

var _ core.Panel = (*EnvelopePanel)(nil)

// NewEnvelopePanel creates an envelope for a tool call.
func NewEnvelopePanel(id, toolName, args string) *EnvelopePanel {
	return &EnvelopePanel{
		BasePanel: core.NewBasePanel(id, 0),
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
	p.SetCollapsed(true) // auto-collapse on result
}

func (p *EnvelopePanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	if msg, ok := msg.(tui.ToolResultMsg); ok {
		p.SetResult(msg.Output, msg.IsError)
	}
	return p, nil
}

func (p *EnvelopePanel) View(width int) string {
	if p.Collapsed() {
		return p.summaryView()
	}
	return p.expandedView(width)
}

func (p *EnvelopePanel) summaryView() string {
	lines := strings.Count(p.output, "\n") + 1
	state := elements.StateDone
	if p.isError {
		state = elements.StateError
	}
	if !p.done {
		state = elements.StateActive
	}
	return "  " + tui.ToolStatus(p.toolName, state, lines)
}

func (p *EnvelopePanel) expandedView(width int) string {
	var sb strings.Builder
	state := elements.StateActive
	if p.done {
		state = elements.StateDone
	}
	if p.isError {
		state = elements.StateError
	}
	fmt.Fprintf(&sb, "  %s %s\n",
		tui.ToolStatus(p.toolName, state, 0),
		tui.ToolArgStyle.Render(p.args))
	if p.output != "" {
		wrapped := tui.WrapText(p.output, width-4)
		for _, line := range strings.Split(wrapped, "\n") {
			sb.WriteString("    " + tui.DimStyle.Render(line) + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}
