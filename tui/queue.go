// queue.go — QueuePanel displays queued prompts between output and input.
// Only visible when the queue is non-empty. Drains top-to-bottom.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Queue panel messages.
type QueueAddMsg struct{ Prompt string }
type QueueDrainMsg struct{}     // remove first item
type QueueClearMsg struct{}     // clear all
type QueueRemoveMsg struct{ Index int }

// QueuePanel displays queued prompts awaiting submission.
type QueuePanel struct {
	BasePanel
	items []string
}

const panelIDQueue = "queue"

var _ Panel = (*QueuePanel)(nil)

func NewQueuePanel() *QueuePanel {
	return &QueuePanel{
		BasePanel: NewBasePanel(panelIDQueue, 0),
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

func (p *QueuePanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case QueueAddMsg:
		p.items = append(p.items, msg.Prompt)
	case QueueDrainMsg:
		if len(p.items) > 0 {
			p.items = p.items[1:]
		}
	case QueueClearMsg:
		p.items = nil
	case QueueRemoveMsg:
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
	sb.WriteString(DimStyle.Render("  queued:"))
	for i, item := range p.items {
		sb.WriteString(fmt.Sprintf("\n  %s %s",
			DimStyle.Render(fmt.Sprintf("%d.", i+1)),
			UserStyle.Render(truncateQueue(item, width-6))))
	}
	return sb.String()
}

func truncateQueue(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
