// budget_panel.go — Live heuristic dashboard during agent work (TSK-484).
//
// Shows Tier 1-4 budget heuristic state as progress bars:
// green (<70%), yellow (70-100%), red (exceeded).
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/review"
	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
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

func (p *BudgetPanel) ID() string             { return p.id }
func (p *BudgetPanel) Focused() bool          { return p.focused }
func (p *BudgetPanel) SetFocus(b bool)        { p.focused = b }
func (p *BudgetPanel) Children() []core.Panel { return nil }
func (p *BudgetPanel) Height() int            { return 0 }
func (p *BudgetPanel) Collapsible() bool      { return true }
func (p *BudgetPanel) Collapsed() bool        { return len(p.signals) == 0 }
func (p *BudgetPanel) Toggle()                {}

// CellSight returns the budget panel's state for agent prompt injection.
// Agent count is public; all cost-related fields are sensitive — budget gaming risk.
func (p *BudgetPanel) CellSight() tui.CellSight {
	fields := []tui.SightField{
		{Key: "signals", Value: fmt.Sprintf("%d", len(p.signals))},
	}
	for i := range p.signals {
		s := &p.signals[i]
		fields = append(fields, tui.SightField{
			Key:       s.Metric,
			Value:     fmt.Sprintf("%.0f/%.0f", s.Value, s.Threshold),
			Sensitive: true,
		})
	}
	return tui.CellSight{
		PanelID: p.id,
		Kind:    "cost",
		Fields:  fields,
	}
}

// SightGate returns false — budget panel is hidden from agents by default.
// The operator must explicitly enable it with :sight reveal.
func (p *BudgetPanel) SightGate() bool { return false }

// Compile-time check.
var _ tui.Sighted = (*BudgetPanel)(nil)

// SetSignals updates the displayed heuristic signals.
func (p *BudgetPanel) SetSignals(signals []review.Signal) {
	p.signals = signals
}

// Update handles messages (no-op for this read-only panel).
func (p *BudgetPanel) Update(_ tea.Msg) (core.Panel, tea.Cmd) {
	return p, nil
}

// View renders the budget dashboard.
func (p *BudgetPanel) View(width int) string {
	if len(p.signals) == 0 {
		return tui.DimStyle.Render("No budget signals")
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
		bar = tui.ErrorStyle.Render(fillChar) + tui.DimStyle.Render(emptyChar)
	case ratio >= 0.7: //nolint:mnd // yellow threshold
		bar = tui.ToolNameStyle.Render(fillChar) + tui.DimStyle.Render(emptyChar)
	default:
		bar = tui.ToolSuccessStyle.Render(fillChar) + tui.DimStyle.Render(emptyChar)
	}

	value := fmt.Sprintf("%.0f/%.0f", s.Value, s.Threshold)
	return label + bar + " " + value
}
