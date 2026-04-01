// thinking.go — ThinkingPanel shows agent thinking during streaming.
// Visible only during streaming. Not focusable. Read-only.
package widgets

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// ThinkingPanel displays the agent's thinking/reasoning text.
type ThinkingPanel struct {
	core.BasePanel
	text   string
	active bool
}

const panelIDThinking = "thinking"

var _ core.Panel = (*ThinkingPanel)(nil)

func NewThinkingPanel() *ThinkingPanel {
	return &ThinkingPanel{
		BasePanel: core.NewBasePanel(panelIDThinking, 1),
	}
}

// Active returns whether thinking is currently displayed.
func (p *ThinkingPanel) Active() bool { return p.active && p.text != "" }

func (p *ThinkingPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.ThinkingMsg:
		p.text = string(msg)
		p.active = true
	case tui.ThinkingClearMsg:
		p.text = ""
		p.active = false
	}
	return p, nil
}

func (p *ThinkingPanel) View(width int) string {
	if !p.active || p.text == "" {
		return ""
	}
	return tui.DimStyle.Render("  " + tui.SpinnerFrames[0] + " " + p.text)
}
