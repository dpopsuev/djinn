// coherence.go — CoherenceGauge renders context window utilization.
// Reads usage from ContextMonitor and renders a visual bar with color zones.
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

const panelIDCoherence = "coherence"

// CoherenceGauge shows context window usage as a colored bar.
type CoherenceGauge struct {
	core.BasePanel
	agentID string
	usage   float64 // 0.0-1.0 from ContextMonitor
}

var _ core.Panel = (*CoherenceGauge)(nil)

// NewCoherenceGauge creates a gauge for the given agent.
func NewCoherenceGauge(agentID string) *CoherenceGauge {
	return &CoherenceGauge{
		BasePanel: core.NewBasePanel(panelIDCoherence, 1),
		agentID:   agentID,
	}
}

// SetUsage updates the context utilization fraction (0.0-1.0).
func (g *CoherenceGauge) SetUsage(usage float64) {
	if usage < 0 {
		usage = 0
	}
	if usage > 1 {
		usage = 1
	}
	g.usage = usage
}

// Zone returns the coherence zone name based on usage.
func (g *CoherenceGauge) Zone() string {
	switch {
	case g.usage < 0.20:
		return "cold"
	case g.usage < 0.40:
		return "warm"
	case g.usage < 0.65:
		return "focused"
	case g.usage < 0.85:
		return "hot"
	default:
		return "redline"
	}
}

func (g *CoherenceGauge) zoneStyle() lipgloss.Style {
	ss := design.ActiveStyles
	switch g.Zone() {
	case "cold":
		return ss.ZoneCold
	case "warm":
		return ss.ZoneWarm
	case "focused":
		return ss.ZoneFocused
	case "hot":
		return ss.ZoneHot
	case "redline":
		return ss.ZoneRedline
	default:
		return tui.DimStyle
	}
}

func (g *CoherenceGauge) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	return g, nil
}

// View renders the coherence gauge as a progress bar with zone label.
func (g *CoherenceGauge) View(width int) string {
	// Reserve space for: label + percentage + zone + padding.
	// Format: "ctx: ████████░░░░░░░░░░░░ 72% focused"
	prefix := "ctx: "
	pct := fmt.Sprintf(" %d%% %s", int(g.usage*100), g.Zone())
	barWidth := width - len(prefix) - len(pct) - 2
	if barWidth < 4 {
		barWidth = 4
	}

	filled := int(float64(barWidth) * g.usage)
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	style := g.zoneStyle()
	bar := style.Render(strings.Repeat("\u2588", filled)) + tui.DimStyle.Render(strings.Repeat("\u2591", empty))

	return prefix + bar + pct
}
