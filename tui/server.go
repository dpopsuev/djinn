// server.go — ServerPanel shows MCP server detail.
// Summary: name + status dot. Drillable into tool list.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// ServerPanel displays MCP server info with drillable tool children.
type ServerPanel struct {
	BasePanel
	name   string
	status HealthStatus
	url    string
	tools  []Panel // child panels for each tool
}

var _ Panel = (*ServerPanel)(nil)

// NewServerPanel creates a server panel.
func NewServerPanel(id, name string, status HealthStatus, url string, tools []Panel) *ServerPanel {
	return &ServerPanel{
		BasePanel: NewBasePanel(id, 1),
		name:      name,
		status:    status,
		url:       url,
		tools:     tools,
	}
}

// Children returns tool panels — enables drill-down into tool list.
func (p *ServerPanel) Children() []Panel { return p.tools }

func (p *ServerPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *ServerPanel) View(width int) string {
	var indicator string
	switch p.status {
	case StatusGreen:
		indicator = healthGreen.Render(GlyphToolSuccess)
	case StatusYellow:
		indicator = healthYellow.Render("⚠")
	case StatusRed:
		indicator = healthRed.Render(GlyphToolError)
	default:
		indicator = fieldKeyStyle.Render("·")
	}
	return fmt.Sprintf("%s %s %s",
		indicator, p.name,
		DimStyle.Render(fmt.Sprintf("(%d tools)", len(p.tools))))
}
