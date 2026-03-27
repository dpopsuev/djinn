// drift.go — DriftPanel renders the three-pillar reconciliation gauge (GOL-31, TSK-278).
//
// Displays functionality, structure, and performance scores as progress bars
// with labels and a convergence counter. Updated via DriftUpdateMsg.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

const panelIDDrift = "drift"

// Pillar color styles — set by ApplyTokens(), never hardcode hex here.
var (
	driftGoodStyle lipgloss.Style
	driftMidStyle  lipgloss.Style
	driftBadStyle  lipgloss.Style
)

// DriftPanel displays three-pillar drift scores with progress bars.
type DriftPanel struct {
	BasePanel
	funcScore float64
	funcLabel string
	archScore float64
	archLabel string
	perfScore float64
	perfLabel string
	tasksLeft int
}

var _ Panel = (*DriftPanel)(nil)

// NewDriftPanel creates a drift panel with zero scores.
func NewDriftPanel() *DriftPanel {
	return &DriftPanel{
		BasePanel: NewBasePanel(panelIDDrift, 4),
	}
}

// SetDrift updates all three pillar scores and the convergence counter.
func (d *DriftPanel) SetDrift(funcScore, archScore, perfScore float64, funcLabel, archLabel, perfLabel string, tasksLeft int) {
	d.funcScore = clampDrift(funcScore)
	d.archScore = clampDrift(archScore)
	d.perfScore = clampDrift(perfScore)
	d.funcLabel = funcLabel
	d.archLabel = archLabel
	d.perfLabel = perfLabel
	d.tasksLeft = tasksLeft
}

func (d *DriftPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	if m, ok := msg.(DriftUpdateMsg); ok {
		d.SetDrift(m.FuncScore, m.ArchScore, m.PerfScore,
			m.FuncLabel, m.ArchLabel, m.PerfLabel, m.TasksLeft)
	}
	return d, nil
}

// View renders the drift panel as three progress bars plus a convergence line.
//
//	Func ████████████████░░░░ 80%  4/5 specs
//	Arch ██████████████████░░ 90%  0 cycles
//	Perf ████████████░░░░░░░░ 60%  3 failing
//	Drift: 3 tasks to convergence
func (d *DriftPanel) View(width int) string {
	var sb strings.Builder
	sb.WriteString(renderPillarBar("Func", d.funcScore, d.funcLabel, width))
	sb.WriteString("\n")
	sb.WriteString(renderPillarBar("Arch", d.archScore, d.archLabel, width))
	sb.WriteString("\n")
	sb.WriteString(renderPillarBar("Perf", d.perfScore, d.perfLabel, width))
	sb.WriteString("\n")
	sb.WriteString(DimStyle.Render(fmt.Sprintf("Drift: %d tasks to convergence", d.tasksLeft)))
	return sb.String()
}

// renderPillarBar renders a single pillar as: "Name ████░░░░ XX%  label"
func renderPillarBar(name string, score float64, label string, width int) string {
	prefix := fmt.Sprintf("%-4s ", name)
	pctStr := fmt.Sprintf(" %3.0f%%", score)
	labelStr := ""
	if label != "" {
		labelStr = "  " + label
	}

	barWidth := width - len(prefix) - len(pctStr) - len(labelStr)
	if barWidth < 4 {
		barWidth = 4
	}

	filled := int(float64(barWidth) * score / 100)
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	style := pillarStyle(score)
	bar := style.Render(strings.Repeat("\u2588", filled)) +
		DimStyle.Render(strings.Repeat("\u2591", empty))

	return prefix + bar + pctStr + DimStyle.Render(labelStr)
}

// pillarStyle returns color based on score threshold.
func pillarStyle(score float64) lipgloss.Style {
	switch {
	case score >= 80:
		return driftGoodStyle
	case score >= 50:
		return driftMidStyle
	default:
		return driftBadStyle
	}
}

// clampDrift clamps a drift score to [0, 100].
func clampDrift(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
