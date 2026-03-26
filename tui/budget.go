// budget.go — BudgetGauge renders cost tracking as a progress bar.
// Shows spent vs ceiling with percentage.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

const panelIDBudget = "budget"

// BudgetGauge shows cost spent vs ceiling as a progress bar.
type BudgetGauge struct {
	BasePanel
	agentID string
	spent   float64
	ceiling float64
}

var _ Panel = (*BudgetGauge)(nil)

// NewBudgetGauge creates a budget gauge with the given ceiling.
func NewBudgetGauge(agentID string, ceiling float64) *BudgetGauge {
	return &BudgetGauge{
		BasePanel: NewBasePanel(panelIDBudget, 1),
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

func (g *BudgetGauge) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return g, nil
}

// Budget bar color styles.
var (
	budgetOKStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})
	budgetWarnStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"})
	budgetOverStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"})
)

func (g *BudgetGauge) budgetStyle() lipgloss.Style {
	if g.ceiling <= 0 {
		return DimStyle
	}
	ratio := g.spent / g.ceiling
	switch {
	case ratio > 0.90:
		return budgetOverStyle
	case ratio > 0.70:
		return budgetWarnStyle
	default:
		return budgetOKStyle
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
	bar := style.Render(strings.Repeat("\u2588", filled)) + DimStyle.Render(strings.Repeat("\u2591", empty))

	return costStr + bar + pctStr
}
