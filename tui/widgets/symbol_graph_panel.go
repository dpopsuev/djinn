// symbol_graph_panel.go — TUI panel showing the pre-edit symbol graph (TSK-650).
//
// Displays the formatted symbol impact table. Implements Sighted for
// agent prompt injection of the currently focused symbol.
package widgets

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	tui "github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/tui/core"
)

// SymbolGraphPanel displays the pre-edit symbol impact table.
type SymbolGraphPanel struct {
	core.BasePanel
	content string // formatted symbol graph text
	file    string // source file path
}

// NewSymbolGraphPanel creates a symbol graph panel.
func NewSymbolGraphPanel() *SymbolGraphPanel {
	return &SymbolGraphPanel{
		BasePanel: core.NewBasePanel("symbol-graph", 0),
	}
}

// SetContent updates the formatted symbol graph text and source file.
func (p *SymbolGraphPanel) SetContent(file, content string) {
	p.file = file
	p.content = content
}

// Update handles messages (no interactivity for now).
func (p *SymbolGraphPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	_ = msg
	return p, nil
}

// View renders the symbol graph text.
func (p *SymbolGraphPanel) View(width int) string {
	if p.content == "" {
		return tui.DimStyle.Render("No symbol graph loaded")
	}
	// Truncate lines to width.
	lines := strings.Split(p.content, "\n")
	var b strings.Builder
	for i, line := range lines {
		if width > 0 && len(line) > width {
			line = line[:width]
		}
		b.WriteString(line)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// CellSight reports the panel's current state for agent prompt injection.
func (p *SymbolGraphPanel) CellSight() tui.CellSight {
	sight := tui.CellSight{
		PanelID: p.ID(),
		Kind:    "symbol-graph",
	}
	if p.file != "" {
		sight.CellID = p.file
		sight.CellTitle = p.file
		sight.Fields = []tui.SightField{
			{Key: "file", Value: p.file},
		}
	}
	return sight
}

// SightGate returns true — symbol graph panel is visible to agents.
func (p *SymbolGraphPanel) SightGate() bool { return true }

// Compile-time checks.
var (
	_ core.Panel  = (*SymbolGraphPanel)(nil)
	_ tui.Sighted = (*SymbolGraphPanel)(nil)
)
