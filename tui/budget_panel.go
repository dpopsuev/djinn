// budget_panel.go — Live heuristic dashboard during agent work (TSK-484).
//
// Shows Tier 1-4 budget heuristic state as progress bars:
// green (<70%), yellow (70-100%), red (exceeded).
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/review"
)

// BudgetPanel displays live backpressure heuristic state.
type BudgetPanel struct {
	id      string
	focused bool
	signals []review.Signal
}

// NewBudgetPanel creates a budget dashboard panel.
func NewBudgetPanel() *BudgetPanel {
	return &BudgetPanel{id: "budget"}
}

func (p *BudgetPanel) ID() string        { return p.id }
func (p *BudgetPanel) Focused() bool     { return p.focused }
func (p *BudgetPanel) SetFocus(b bool)   { p.focused = b }
func (p *BudgetPanel) Children() []Panel { return nil }
func (p *BudgetPanel) Height() int       { return 0 }
func (p *BudgetPanel) Collapsible() bool { return true }
func (p *BudgetPanel) Collapsed() bool   { return len(p.signals) == 0 }
func (p *BudgetPanel) Toggle()           {}

// SetSignals updates the displayed heuristic signals.
func (p *BudgetPanel) SetSignals(signals []review.Signal) {
	p.signals = signals
}

// Update handles messages (no-op for this read-only panel).
func (p *BudgetPanel) Update(_ tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

// View renders the budget dashboard.
func (p *BudgetPanel) View(width int) string {
	if len(p.signals) == 0 {
		return DimStyle.Render("No budget signals")
	}

	var b strings.Builder
	for i := range p.signals {
		s := &p.signals[i]
		if s.Threshold <= 0 {
			continue
		}
		line := renderBar(s, width-4) //nolint:mnd // padding
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}

func renderBar(s *review.Signal, maxWidth int) string {
	label := padRight(s.Metric, 20) //nolint:mnd // label column
	barWidth := maxWidth - 30       //nolint:mnd // space for label + value
	if barWidth < 10 {              //nolint:mnd // minimum bar width
		barWidth = 10
	}

	ratio := s.Value / s.Threshold
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(barWidth))
	empty := barWidth - filled

	// Color based on ratio.
	var bar string
	fillChar := strings.Repeat("█", filled)
	emptyChar := strings.Repeat("░", empty)

	switch {
	case s.Exceeded:
		bar = ErrorStyle.Render(fillChar) + DimStyle.Render(emptyChar)
	case ratio >= 0.7: //nolint:mnd // yellow threshold
		bar = ToolNameStyle.Render(fillChar) + DimStyle.Render(emptyChar)
	default:
		bar = ToolSuccessStyle.Render(fillChar) + DimStyle.Render(emptyChar)
	}

	value := fmt.Sprintf("%.0f/%.0f", s.Value, s.Threshold)
	return label + bar + " " + value
}
