// dashboard.go — DashboardPanel renders the status line with health widgets.
package widgets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/tui/core"
	"github.com/dpopsuev/djinn/tui/design"

	tui "github.com/dpopsuev/djinn/tui"
)

// DashboardPanel is the bottom status line.
type DashboardPanel struct {
	core.BasePanel
	workspace  string
	driver     string
	model      string
	mode       string
	tokensIn   int
	tokensOut  int
	turns      int
	agentCount int
	activeRole string
	operation  string // ask/plan/agent
	agentCap   int    // max concurrent agents
	health     []tui.HealthReport
	uiState    string // "INSERT", "STREAMING", "APPROVAL"
}

const panelIDDashboard = "dashboard"

var _ core.Panel = (*DashboardPanel)(nil)

// NewDashboardPanel creates the dashboard.
func NewDashboardPanel() *DashboardPanel {
	return &DashboardPanel{
		BasePanel: core.NewBasePanel(panelIDDashboard, 1),
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
func (p *DashboardPanel) SetHealth(reports []tui.HealthReport) {
	p.health = reports
}

// SetUIState sets the vim-style mode indicator.
func (p *DashboardPanel) SetUIState(state string) {
	p.uiState = state
}

func (p *DashboardPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.DashboardIdentityMsg:
		p.workspace = msg.Workspace
		p.driver = msg.Driver
		p.model = msg.Model
		p.mode = msg.Mode
	case tui.DashboardMetricsMsg:
		p.tokensIn = msg.TokensIn
		p.tokensOut = msg.TokensOut
		p.turns = msg.Turns
		p.agentCount = msg.AgentCount
		p.activeRole = msg.ActiveRole
		p.operation = msg.Operation
		p.agentCap = msg.AgentCap
	case tui.DashboardHealthMsg:
		p.health = msg.Reports
	case tui.DashboardUIStateMsg:
		p.uiState = msg.State
	}
	return p, nil
}

func (p *DashboardPanel) View(width int) string {
	ss := design.ActiveStyles

	// Left: vim-style mode indicator.
	var indicator string
	switch p.uiState {
	case "STREAMING":
		indicator = ss.ModeStream.Render("-- STREAMING --")
	case "APPROVAL":
		indicator = ss.ModeApproval.Render("-- APPROVAL --")
	default:
		indicator = ss.ModeInsert.Render("-- INSERT --")
	}

	// Right: status fields.
	statusLine := tui.RenderStatusLine(p.workspace, p.driver, p.model, p.mode,
		p.tokensIn, p.tokensOut, p.turns, p.health)

	// Operation indicator (only if set).
	if p.operation != "" {
		opInfo := tui.StatusStyle.Render(fmt.Sprintf("[%s] ", p.operation))
		statusLine = opInfo + statusLine
	}

	// Multi-agent: prepend agent count/capacity to status line.
	if p.agentCap > 0 {
		agentInfo := tui.StatusStyle.Render(fmt.Sprintf("agents:%d/%d ", p.agentCount, p.agentCap))
		if p.activeRole != "" {
			agentInfo = tui.StatusStyle.Render(fmt.Sprintf("agents:%d/%d active:%s ", p.agentCount, p.agentCap, p.activeRole))
		}
		statusLine = agentInfo + statusLine
	} else if p.agentCount > 1 {
		agentInfo := tui.StatusStyle.Render(fmt.Sprintf("agents:%d active:%s | ", p.agentCount, p.activeRole))
		statusLine = agentInfo + statusLine
	}

	// Compose: indicator left, status right, fill middle with spaces.
	gap := width - lipgloss.Width(indicator) - lipgloss.Width(statusLine)
	if gap < 1 {
		gap = 1
	}
	return fmt.Sprintf("  %s%*s%s", indicator, gap, "", statusLine)
}
