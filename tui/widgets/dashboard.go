// dashboard.go — DashboardPanel renders the status line with health widgets.
package widgets

import (
	"context"
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/djinnlog"
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
	andon      tui.AndonState
}

const panelIDDashboard = "dashboard"

var _ core.Panel = (*DashboardPanel)(nil)
var _ tui.Sighted = (*DashboardPanel)(nil)

// DashboardOption configures a DashboardPanel.
type DashboardOption func(*DashboardPanel)

// WithWorkspace sets the workspace name shown in the dashboard.
func WithWorkspace(name string) DashboardOption {
	return func(p *DashboardPanel) { p.workspace = name }
}

// WithDriverModel sets the driver and model shown in the dashboard.
func WithDriverModel(driver, model string) DashboardOption {
	return func(p *DashboardPanel) { p.driver = driver; p.model = model }
}

// WithUIState sets the initial UI state indicator (INSERT, STREAMING, etc.).
func WithUIState(state string) DashboardOption {
	return func(p *DashboardPanel) { p.uiState = state }
}

// NewDashboardPanel creates the dashboard.
func NewDashboardPanel(opts ...DashboardOption) *DashboardPanel {
	p := &DashboardPanel{
		BasePanel: core.NewBasePanel(panelIDDashboard, 1),
		uiState:   "INSERT",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
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

// CellSight returns the dashboard's current state for agent prompt injection.
// Tokens are sensitive — an agent could game budget by seeing remaining capacity.
func (p *DashboardPanel) CellSight() tui.CellSight {
	return tui.CellSight{
		PanelID: panelIDDashboard,
		Kind:    "status",
		Fields: []tui.SightField{
			{Key: "operation", Value: p.operation},
			{Key: "turns", Value: fmt.Sprintf("%d", p.turns)},
			{Key: "agents", Value: fmt.Sprintf("%d", p.agentCount)},
			{Key: "tokens", Value: fmt.Sprintf("%d/%d", p.tokensIn, p.tokensOut), Sensitive: true},
		},
	}
}

// SightGate returns true — dashboard is visible to agents by default.
func (p *DashboardPanel) SightGate() bool { return true }

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
	case tui.AndonUpdateMsg:
		prev := p.andon.Level
		p.andon = msg.State
		if msg.State.Level != prev {
			slog.InfoContext(context.Background(), "andon level change",
				slog.String(djinnlog.KeyComponent, "dashboard"),
				slog.String(djinnlog.KeyFrom, prev.String()),
				slog.String(djinnlog.KeyTo, msg.State.Level.String()),
				slog.String(djinnlog.KeyAgent, msg.State.Source),
			)
		}
		if tui.ShouldCordon(msg.State.Level) {
			slog.WarnContext(context.Background(), "cordon triggered by andon",
				slog.String(djinnlog.KeyComponent, "dashboard"),
				slog.String(djinnlog.KeyAgent, msg.State.Source),
				slog.String(djinnlog.KeyReason, msg.State.Message),
			)
			return p, func() tea.Msg {
				return tui.CordonMsg{
					Reason:  msg.State.Message,
					AgentID: msg.State.Source,
					Detail:  "andon red: auto-cordon",
				}
			}
		}
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

	// Andon indicator.
	var andonStr string
	switch p.andon.Level {
	case tui.AndonGreen:
		andonStr = ss.HealthGreen.Render("\u25cf") // ●
	case tui.AndonYellow:
		andonStr = ss.HealthYellow.Render("\u25c9") // ◉
	case tui.AndonRed:
		andonStr = ss.HealthRed.Render("\u2b24") // ⬤
	}
	statusLine = andonStr + " " + statusLine

	// Compose: indicator left, status right, fill middle with spaces.
	gap := width - lipgloss.Width(indicator) - lipgloss.Width(statusLine)
	if gap < 1 {
		gap = 1
	}
	return fmt.Sprintf("  %s%*s%s", indicator, gap, "", statusLine)
}
