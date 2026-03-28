// thinking.go — ThinkingPanel shows agent thinking during streaming.
// Visible only during streaming. Not focusable. Read-only.
package tui

import tea "github.com/charmbracelet/bubbletea"

// ThinkingPanel displays the agent's thinking/reasoning text.
type ThinkingPanel struct {
	BasePanel
	text   string
	active bool
}

const panelIDThinking = "thinking"

var _ Panel = (*ThinkingPanel)(nil)

func NewThinkingPanel() *ThinkingPanel {
	return &ThinkingPanel{
		BasePanel: NewBasePanel(panelIDThinking, 1),
	}
}

// Active returns whether thinking is currently displayed.
func (p *ThinkingPanel) Active() bool { return p.active && p.text != "" }

func (p *ThinkingPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case ThinkingMsg:
		p.text = string(msg)
		p.active = true
	case ThinkingClearMsg:
		p.text = ""
		p.active = false
	}
	return p, nil
}

func (p *ThinkingPanel) View(width int) string {
	if !p.active || p.text == "" {
		return ""
	}
	return DimStyle.Render("  " + SpinnerFrames[0] + " " + p.text)
}
