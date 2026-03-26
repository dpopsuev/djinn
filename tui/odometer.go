// odometer.go — OdometerLine renders session summary stats in a single line.
// Shows round-trips, cost, tasks completed, and relay count.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

const panelIDOdometer = "odometer"

// OdometerLine shows cumulative session stats.
type OdometerLine struct {
	BasePanel
	roundTrips int
	cost       float64
	tasks      int
	relays     int
}

var _ Panel = (*OdometerLine)(nil)

// NewOdometerLine creates an odometer with zeroed counters.
func NewOdometerLine() *OdometerLine {
	return &OdometerLine{
		BasePanel: NewBasePanel(panelIDOdometer, 1),
	}
}

// Update handles OdometerUpdateMsg.
func (o *OdometerLine) Update(msg tea.Msg) (Panel, tea.Cmd) {
	if m, ok := msg.(OdometerUpdateMsg); ok {
		o.roundTrips = m.RoundTrips
		o.cost = m.Cost
		o.tasks = m.Tasks
		o.relays = m.Relays
	}
	return o, nil
}

// View renders the odometer as a single status line.
// Format: round-trips:247 | $6.77 spent | 14 tasks | 3 relays
func (o *OdometerLine) View(width int) string {
	return fmt.Sprintf("round-trips:%d | $%.2f spent | %d tasks | %d relays",
		o.roundTrips, o.cost, o.tasks, o.relays)
}
