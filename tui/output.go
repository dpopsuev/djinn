// output.go — OutputPanel wraps a viewport for scrollable conversation.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// OutputPanel is the scrollable conversation area.
type OutputPanel struct {
	BasePanel
	vp       viewport.Model
	vpReady  bool
	lines    []string
}

// NewOutputPanel creates the output panel.
func NewOutputPanel() *OutputPanel {
	return &OutputPanel{
		BasePanel: NewBasePanel("output", 0), // flex height
	}
}

func (p *OutputPanel) InitViewport(width, height int) {
	if height < 3 {
		height = 3
	}
	if !p.vpReady {
		p.vp = viewport.New(width, height)
		p.vpReady = true
	} else {
		p.vp.Width = width
		p.vp.Height = height
	}
	p.syncViewport()
}

// Append adds a line to the output.
func (p *OutputPanel) Append(line string) {
	p.lines = append(p.lines, line)
	p.syncViewport()
}

// SetLine replaces a specific line by index.
func (p *OutputPanel) SetLine(idx int, line string) {
	if idx >= 0 && idx < len(p.lines) {
		p.lines[idx] = line
		p.syncViewport()
	}
}

// LineCount returns the number of lines.
func (p *OutputPanel) LineCount() int {
	return len(p.lines)
}

// Lines returns all lines.
func (p *OutputPanel) Lines() []string {
	return p.lines
}

// Clear removes all lines.
func (p *OutputPanel) Clear() {
	p.lines = nil
	p.syncViewport()
}

func (p *OutputPanel) syncViewport() {
	if p.vpReady {
		p.vp.SetContent(strings.Join(p.lines, "\n"))
		p.vp.GotoBottom()
	}
}

func (p *OutputPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	if !p.focused || !p.vpReady {
		return p, nil
	}
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return p, cmd
}

func (p *OutputPanel) View(width int) string {
	if !p.vpReady {
		return strings.Join(p.lines, "\n")
	}
	return p.vp.View()
}
