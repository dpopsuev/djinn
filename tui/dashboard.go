// dashboard.go — DashboardPanel renders the status line with health widgets.
package tui

import tea "github.com/charmbracelet/bubbletea"

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
}

const panelIDDashboard = "dashboard"

var _ Panel = (*DashboardPanel)(nil)

// NewDashboardPanel creates the dashboard.
func NewDashboardPanel() *DashboardPanel {
	return &DashboardPanel{
		BasePanel: NewBasePanel(panelIDDashboard, 1),
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

func (p *DashboardPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *DashboardPanel) View(width int) string {
	return RenderStatusLine(p.workspace, p.driver, p.model, p.mode,
		p.tokensIn, p.tokensOut, p.turns, p.health)
}
