// tour_panel.go — TUI panel for circuit-based visual review (TSK-446).
//
// Two-level navigation: Tab between circuits, j/k between stops.
// Implements the Panel interface for integration with LayoutEngine.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/render"
	"github.com/dpopsuev/djinn/review"
)

// TourPanel displays circuit-based review tours.
type TourPanel struct {
	id       string
	tour     *review.TourView
	focused  bool
	circuit  int // current circuit index
	stop     int // current stop index
	renderer render.Renderer
	width    int
}

// NewTourPanel creates a tour panel with TUI rendering.
func NewTourPanel() *TourPanel {
	return &TourPanel{
		id:       "tour",
		circuit:  0,
		stop:     0,
		renderer: render.NewTUIRenderer(),
	}
}

func (p *TourPanel) ID() string        { return p.id }
func (p *TourPanel) Focused() bool     { return p.focused }
func (p *TourPanel) SetFocus(b bool)   { p.focused = b }
func (p *TourPanel) Children() []Panel { return nil }
func (p *TourPanel) Height() int       { return 0 } // flex
func (p *TourPanel) Collapsible() bool { return true }
func (p *TourPanel) Collapsed() bool   { return p.tour == nil }
func (p *TourPanel) Toggle()           {}

// SetTour replaces the current tour data.
func (p *TourPanel) SetTour(tour *review.TourView) {
	p.tour = tour
	p.circuit = 0
	p.stop = 0
}

// Update handles keyboard navigation.
func (p *TourPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	if !p.focused || p.tour == nil {
		return p, nil
	}

	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch km.String() {
	case "j", "down":
		p.nextStop()
	case "k", "up":
		p.prevStop()
	case "tab":
		p.nextCircuit()
	case "shift+tab":
		p.prevCircuit()
	}

	return p, nil
}

// View renders the current circuit's stops.
func (p *TourPanel) View(width int) string {
	p.width = width
	if p.tour == nil || len(p.tour.Circuits) == 0 {
		return DimStyle.Render("No circuits to display. Run :tour to generate.")
	}

	var b strings.Builder

	// Circuit selector.
	fmt.Fprintf(&b, "Circuit %d/%d: %s\n",
		p.circuit+1, len(p.tour.Circuits), p.tour.Circuits[p.circuit].Title)

	// Stops.
	cv := p.tour.Circuits[p.circuit]
	for i, sv := range cv.Stops {
		prefix := "  "
		if i == p.stop {
			prefix = "▸ "
		}

		name := sv.Name
		if sv.Changed {
			name = ToolSuccessStyle.Render("● ") + name
		} else if sv.PassThrough {
			name = DimStyle.Render("○ " + name)
		}

		b.WriteString(prefix + name)
		if sv.Detail != "" {
			b.WriteString(" — " + DimStyle.Render(sv.Detail))
		}
		if sv.CrossRef != "" {
			b.WriteString(" " + DimStyle.Render("["+sv.CrossRef+"]"))
		}
		b.WriteByte('\n')
	}

	// Dead code warning.
	if len(p.tour.DeadCode) > 0 {
		fmt.Fprintf(&b, "\n⚠ %d unreachable changes (dead code)\n", len(p.tour.DeadCode))
	}

	return b.String()
}

func (p *TourPanel) nextStop() {
	if p.tour == nil || len(p.tour.Circuits) == 0 {
		return
	}
	cv := p.tour.Circuits[p.circuit]
	if p.stop < len(cv.Stops)-1 {
		p.stop++
	}
}

func (p *TourPanel) prevStop() {
	if p.stop > 0 {
		p.stop--
	}
}

func (p *TourPanel) nextCircuit() {
	if p.tour == nil {
		return
	}
	if p.circuit < len(p.tour.Circuits)-1 {
		p.circuit++
		p.stop = 0
	}
}

func (p *TourPanel) prevCircuit() {
	if p.circuit > 0 {
		p.circuit--
		p.stop = 0
	}
}
