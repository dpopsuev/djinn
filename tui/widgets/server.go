// server.go — ServerPanel shows MCP server detail.
// Summary: name + status dot. Drillable into tool list.
package widgets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// ServerPanel displays MCP server info with drillable tool children.
type ServerPanel struct {
	core.BasePanel
	name   string
	status tui.HealthStatus
	url    string
	tools  []core.Panel // child panels for each tool
}

var _ core.Panel = (*ServerPanel)(nil)

// NewServerPanel creates a server panel.
func NewServerPanel(id, name string, status tui.HealthStatus, url string, tools []core.Panel) *ServerPanel {
	return &ServerPanel{
		BasePanel: core.NewBasePanel(id, 1),
		name:      name,
		status:    status,
		url:       url,
		tools:     tools,
	}
}

// Children returns tool panels — enables drill-down into tool list.
func (p *ServerPanel) Children() []core.Panel { return p.tools }

func (p *ServerPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	return p, nil
}

func (p *ServerPanel) View(width int) string {
	var indicator string
	switch p.status {
	case tui.StatusGreen:
		indicator = tui.Glyph(tui.StateDone)
	case tui.StatusYellow:
		indicator = tui.Glyph(tui.StatePending)
	case tui.StatusRed:
		indicator = tui.Glyph(tui.StateError)
	default:
		indicator = tui.DimStyle.Render("·")
	}
	return fmt.Sprintf("%s %s %s",
		indicator, p.name,
		tui.DimStyle.Render(fmt.Sprintf("(%d tools)", len(p.tools))))
}
