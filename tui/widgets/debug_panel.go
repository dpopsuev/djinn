// debug_panel.go — Live MCP debugger panel (TSK-481).
//
// Shows real-time trace event stream with component, action, server,
// tool, latency, and error status. Two views: stream (chronological)
// and tree (round-trip correlation by parent ID).
package widgets

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// DebugPanel displays the live trace event stream.
type DebugPanel struct {
	id      string
	ring    *telemetry.TraceProjection
	focused bool
	scroll  int // offset from newest (0 = bottom)
	limit   int // max events to display
}

// NewDebugPanel creates a debug panel backed by a trace ring.
func NewDebugPanel(ring *telemetry.TraceProjection) *DebugPanel {
	return &DebugPanel{
		id:    "debug",
		ring:  ring,
		limit: 20, //nolint:mnd // reasonable default for visible events
	}
}

func (p *DebugPanel) ID() string             { return p.id }
func (p *DebugPanel) Focused() bool          { return p.focused }
func (p *DebugPanel) SetFocus(b bool)        { p.focused = b }
func (p *DebugPanel) Children() []core.Panel { return nil }
func (p *DebugPanel) Height() int            { return 0 } // flex
func (p *DebugPanel) Collapsible() bool      { return true }
func (p *DebugPanel) Collapsed() bool        { return p.ring == nil }
func (p *DebugPanel) Toggle()                {}

// Update handles keyboard navigation.
func (p *DebugPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	if !p.focused {
		return p, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}
	switch km.String() {
	case "j", "down":
		if p.scroll > 0 {
			p.scroll--
		}
	case "k", "up":
		p.scroll++
	case "G":
		p.scroll = 0 // jump to newest
	case "g":
		stats := p.ring.Stats()
		p.scroll = stats.Count - p.limit // jump to oldest
		if p.scroll < 0 {
			p.scroll = 0
		}
	}
	return p, nil
}

// View renders the trace event stream.
func (p *DebugPanel) View(width int) string {
	if p.ring == nil {
		return tui.DimStyle.Render("No trace ring attached")
	}

	events := p.ring.Last(p.limit + p.scroll)
	if p.scroll > 0 && len(events) > p.limit {
		events = events[:len(events)-p.scroll]
	}
	if len(events) > p.limit {
		events = events[len(events)-p.limit:]
	}

	if len(events) == 0 {
		return tui.DimStyle.Render("No trace events yet")
	}

	var b strings.Builder
	for i := range events {
		e := &events[i]
		line := formatTraceEvent(e, width)
		b.WriteString(line)
		b.WriteByte('\n')
	}

	stats := p.ring.Stats()
	b.WriteString(tui.DimStyle.Render(fmt.Sprintf("  %d/%d events", stats.Count, stats.Capacity)))

	return b.String()
}

func formatTraceEvent(e *telemetry.TraceEvent, _ int) string {
	comp := padRight(string(e.Component), 6)
	action := padRight(e.Action, 7)

	// Server + tool or detail.
	info := e.Detail
	if e.Server != "" && e.Tool != "" {
		info = e.Server + "  " + e.Tool
	}
	info = padRight(info, 30) //nolint:mnd // column width for readability

	// Latency.
	lat := "     "
	if e.Latency > 0 {
		lat = padLeft(formatDuration(e.Latency), 5) //nolint:mnd // column width
	}

	// Status.
	status := " "
	if e.Error {
		status = tui.ErrorStyle.Render("✗")
	} else if e.Action == "result" {
		status = tui.ToolSuccessStyle.Render("✓")
	}

	return fmt.Sprintf("  %s %s %s %s %s", comp, action, info, lat, status)
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
}

// CellSight returns the currently focused trace event's context.
// The focused event is the newest visible event (bottom of scroll window).
// Latency and error fields are marked Sensitive — excluded from agent prompt
// unless the operator explicitly reveals them.
func (p *DebugPanel) CellSight() tui.CellSight {
	if p.ring == nil {
		return tui.CellSight{PanelID: p.id}
	}

	events := p.ring.Last(p.limit + p.scroll)
	if p.scroll > 0 && len(events) > p.limit {
		events = events[:len(events)-p.scroll]
	}
	if len(events) == 0 {
		return tui.CellSight{PanelID: p.id}
	}

	// Focus is on the newest visible event.
	e := &events[len(events)-1]
	fields := []tui.SightField{
		{Key: "component", Value: string(e.Component)},
		{Key: "action", Value: e.Action},
	}
	if e.Server != "" {
		fields = append(fields, tui.SightField{Key: "server", Value: e.Server})
	}
	if e.Tool != "" {
		fields = append(fields, tui.SightField{Key: "tool", Value: e.Tool})
	}
	if e.Latency > 0 {
		fields = append(fields, tui.SightField{Key: "latency", Value: formatDuration(e.Latency), Sensitive: true})
	}
	if e.Error {
		fields = append(fields, tui.SightField{Key: "error", Value: "true", Sensitive: true})
	}

	return tui.CellSight{
		PanelID:   p.id,
		CellID:    e.ID,
		CellTitle: e.Detail,
		Kind:      string(e.Component),
		Fields:    fields,
	}
}

// SightGate returns true — debug panel is visible to agents by default.
func (p *DebugPanel) SightGate() bool { return true }

// Compile-time check.
var _ tui.Sighted = (*DebugPanel)(nil)

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
