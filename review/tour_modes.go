// tour_modes.go — Tour mode strategies for visual review (TSK-444).
//
// Each mode formats the same circuit data differently:
// summary (topology only), signatures (before/after), io (input→output),
// impact (callers + blast radius), diagram (render via Rendering Kit).
package review

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/render"
)

// TourMode controls how circuit stops are presented.
type TourMode string

const (
	ModeSummary    TourMode = "summary"    // topology only
	ModeSignatures TourMode = "signatures" // before/after sigs
	ModeIO         TourMode = "io"         // input→output types
	ModeImpact     TourMode = "impact"     // callers + blast radius
	ModeDiagram    TourMode = "diagram"    // render via Rendering Kit
)

// TourView is the formatted output of a tour.
type TourView struct {
	Circuits []CircuitView `json:"circuits"`
	DeadCode []string      `json:"dead_code,omitempty"`
}

// CircuitView is a single circuit formatted for display.
type CircuitView struct {
	Title string     `json:"title"`
	Stops []StopView `json:"stops"`
}

// StopView is a single stop formatted by the current mode.
type StopView struct {
	Name        string `json:"name"`
	Detail      string `json:"detail"`
	Changed     bool   `json:"changed"`
	PassThrough bool   `json:"pass_through"`
	CrossRef    string `json:"cross_ref,omitempty"` // "also on Circuit N"
}

// FormatTour formats circuits using the specified mode.
func FormatTour(circuits []Circuit, mode TourMode) *TourView {
	view := &TourView{
		Circuits: make([]CircuitView, 0, len(circuits)),
	}

	for i := range circuits {
		cv := CircuitView{
			Title: circuits[i].Title,
			Stops: make([]StopView, 0, len(circuits[i].Stops)),
		}
		for j := range circuits[i].Stops {
			s := &circuits[i].Stops[j]
			sv := StopView{
				Name:        qualifiedName(s.Package, s.Name),
				Changed:     s.Changed,
				PassThrough: s.PassThrough,
				Detail:      formatStopDetail(s, mode),
			}
			// Cross-reference shared stops.
			for _, shared := range circuits[i].Shared {
				if shared == s.ID {
					sv.CrossRef = "shared across circuits"
					break
				}
			}
			cv.Stops = append(cv.Stops, sv)
		}
		view.Circuits = append(view.Circuits, cv)
	}

	return view
}

func formatStopDetail(s *render.CircuitStop, mode TourMode) string {
	switch mode {
	case ModeSummary:
		return ""
	case ModeSignatures:
		if s.SignatureBefore != "" && s.SignatureAfter != "" && s.SignatureBefore != s.SignatureAfter {
			return fmt.Sprintf("was: %s → now: %s", s.SignatureBefore, s.SignatureAfter)
		}
		if s.SignatureAfter != "" {
			return s.SignatureAfter
		}
		return s.SignatureBefore
	case ModeIO:
		parts := make([]string, 0, 2) //nolint:mnd // IO has 2 parts
		if s.InputType != "" {
			parts = append(parts, "in: "+s.InputType)
		}
		if s.OutputType != "" {
			parts = append(parts, "out: "+s.OutputType)
		}
		return strings.Join(parts, " → ")
	case ModeImpact:
		if s.Changed {
			return "CHANGED"
		}
		return "pass-through"
	case ModeDiagram:
		return "" // diagram mode uses Rendering Kit, not text
	default:
		return ""
	}
}

func qualifiedName(pkg, name string) string {
	if pkg != "" {
		return pkg + "." + name
	}
	return name
}

// ParseTourMode converts a string argument to TourMode.
func ParseTourMode(args string) TourMode {
	switch strings.TrimSpace(strings.ToLower(args)) {
	case "sigs", "signatures":
		return ModeSignatures
	case "io":
		return ModeIO
	case "impact":
		return ModeImpact
	case "diagram", "d":
		return ModeDiagram
	default:
		return ModeSummary
	}
}
