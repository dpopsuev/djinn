// queue.go — QueuePanel displays queued prompts between output and input.
// Only visible when the queue is non-empty. Drains top-to-bottom.
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// QueuePanel displays queued prompts awaiting submission.
type QueuePanel struct {
	core.BasePanel
	items []string
}

const panelIDQueue = "queue"

var _ core.Panel = (*QueuePanel)(nil)

func NewQueuePanel() *QueuePanel {
	return &QueuePanel{
		BasePanel: core.NewBasePanel(panelIDQueue, 0),
	}
}

// Items returns the current queue contents.
func (p *QueuePanel) Items() []string {
	return p.items
}

// Len returns the number of queued items.
func (p *QueuePanel) Len() int {
	return len(p.items)
}

func (p *QueuePanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.QueueAddMsg:
		p.items = append(p.items, msg.Prompt)
	case tui.QueueDrainMsg:
		if len(p.items) > 0 {
			p.items = p.items[1:]
		}
	case tui.QueueClearMsg:
		p.items = nil
	case tui.QueueRemoveMsg:
		if msg.Index >= 0 && msg.Index < len(p.items) {
			p.items = append(p.items[:msg.Index], p.items[msg.Index+1:]...)
		}
	}
	return p, nil
}

func (p *QueuePanel) View(width int) string {
	if len(p.items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(tui.DimStyle.Render("  queued:"))
	for i, item := range p.items {
		sb.WriteString(fmt.Sprintf("\n  %s %s",
			tui.DimStyle.Render(fmt.Sprintf("%d.", i+1)),
			tui.UserStyle.Render(truncateQueue(item, width-6))))
	}
	return sb.String()
}

func truncateQueue(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
