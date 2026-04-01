// budget.go — BudgetGauge renders cost tracking as a progress bar.
// Shows spent vs ceiling with percentage.
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/tui/core"
	"github.com/dpopsuev/djinn/tui/design"

	tui "github.com/dpopsuev/djinn/tui"
)

const panelIDBudget = "budget"

// BudgetGauge shows cost spent vs ceiling as a progress bar.
type BudgetGauge struct {
	core.BasePanel
	agentID string
	spent   float64
	ceiling float64
}

var _ core.Panel = (*BudgetGauge)(nil)

// NewBudgetGauge creates a budget gauge with the given ceiling.
func NewBudgetGauge(agentID string, ceiling float64) *BudgetGauge {
	return &BudgetGauge{
		BasePanel: core.NewBasePanel(panelIDBudget, 1),
		agentID:   agentID,
		ceiling:   ceiling,
	}
}

// SetSpent updates the amount spent.
func (g *BudgetGauge) SetSpent(spent float64) {
	if spent < 0 {
		spent = 0
	}
	g.spent = spent
}

func (g *BudgetGauge) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	return g, nil
}

func (g *BudgetGauge) budgetStyle() lipgloss.Style {
	ss := design.ActiveStyles
	if g.ceiling <= 0 {
		return tui.DimStyle
	}
	ratio := g.spent / g.ceiling
	switch {
	case ratio > 0.90:
		return ss.BudgetOver
	case ratio > 0.70:
		return ss.BudgetWarn
	default:
		return ss.BudgetOK
	}
}

// View renders the budget gauge.
// Format: $2.40/$10.00 ████████████████░░░░ 24%
func (g *BudgetGauge) View(width int) string {
	ratio := 0.0
	if g.ceiling > 0 {
		ratio = g.spent / g.ceiling
		if ratio > 1 {
			ratio = 1
		}
	}
	pct := int(ratio * 100)

	costStr := fmt.Sprintf("$%.2f/$%.2f ", g.spent, g.ceiling)
	pctStr := fmt.Sprintf(" %d%%", pct)
	barWidth := width - len(costStr) - len(pctStr) - 2
	if barWidth < 4 {
		barWidth = 4
	}

	filled := int(float64(barWidth) * ratio)
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	style := g.budgetStyle()
	bar := style.Render(strings.Repeat("\u2588", filled)) + tui.DimStyle.Render(strings.Repeat("\u2591", empty))

	return costStr + bar + pctStr
}
