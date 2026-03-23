// dashboard.go — DashboardPanel renders the status line with health widgets.
package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// DashboardPanel is the bottom status line.
type DashboardPanel struct {
	BasePanel
	workspace string
	driver    string
	model     string
	mode      string
	tokensIn  int
	tokensOut int
	turns     int
	health    []HealthReport
	uiState   string // "INSERT", "STREAMING", "APPROVAL"
}

const panelIDDashboard = "dashboard"

var _ Panel = (*DashboardPanel)(nil)

// NewDashboardPanel creates the dashboard.
func NewDashboardPanel() *DashboardPanel {
	return &DashboardPanel{
		BasePanel: NewBasePanel(panelIDDashboard, 1),
		uiState:   "INSERT",
	}
}

// SetIdentity updates workspace/driver/model/mode.
func (p *DashboardPanel) SetIdentity(workspace, driver, model, mode string) {
	p.workspace = workspace
	p.driver = driver
	p.model = model
	p.mode = mode
}

// SetMetrics updates token and turn counts.
func (p *DashboardPanel) SetMetrics(tokensIn, tokensOut, turns int) {
	p.tokensIn = tokensIn
	p.tokensOut = tokensOut
	p.turns = turns
}

// SetHealth updates MCP health reports.
func (p *DashboardPanel) SetHealth(reports []HealthReport) {
	p.health = reports
}

// SetUIState sets the vim-style mode indicator.
func (p *DashboardPanel) SetUIState(state string) {
	p.uiState = state
}

func (p *DashboardPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

// Vim-style mode indicator styles.
var (
	modeInsertStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})
	modeStreamStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"})
	modeApprovalStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"})
)

func (p *DashboardPanel) View(width int) string {
	// Left: vim-style mode indicator.
	var indicator string
	switch p.uiState {
	case "STREAMING":
		indicator = modeStreamStyle.Render("-- STREAMING --")
	case "APPROVAL":
		indicator = modeApprovalStyle.Render("-- APPROVAL --")
	default:
		indicator = modeInsertStyle.Render("-- INSERT --")
	}

	// Right: status fields.
	statusLine := RenderStatusLine(p.workspace, p.driver, p.model, p.mode,
		p.tokensIn, p.tokensOut, p.turns, p.health)

	// Compose: indicator left, status right, fill middle with spaces.
	gap := width - lipgloss.Width(indicator) - lipgloss.Width(statusLine)
	if gap < 1 {
		gap = 1
	}
	return fmt.Sprintf("  %s%*s%s", indicator, gap, "", statusLine)
}
