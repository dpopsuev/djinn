// scratch_panel.go — Scratch paper visualization for workstation state (TSK-656).
//
// Renders the four scratch paper sections: understanding, plan, risks, notes.
// Implements Sighted so agents can see what the operator is reviewing.
package widgets

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

// ScratchPaperPanel renders scratch paper sections for the active workstation.
type ScratchPaperPanel struct {
	core.BasePanel
	understanding string
	plan          []string
	risks         []string
	notes         string
}

// NewScratchPaperPanel creates an empty scratch paper panel.
func NewScratchPaperPanel() *ScratchPaperPanel {
	return &ScratchPaperPanel{
		BasePanel: core.NewBasePanel(artifact.KindScratchPaper, 0),
	}
}

// SetUnderstanding updates the understanding section.
func (p *ScratchPaperPanel) SetUnderstanding(text string) {
	p.understanding = text
}

// SetPlan updates the plan section.
func (p *ScratchPaperPanel) SetPlan(steps []string) {
	p.plan = steps
}

// SetRisks updates the risks section.
func (p *ScratchPaperPanel) SetRisks(risks []string) {
	p.risks = risks
}

// SetNotes updates the notes section.
func (p *ScratchPaperPanel) SetNotes(text string) {
	p.notes = text
}

// View renders the scratch paper sections.
func (p *ScratchPaperPanel) View(_ int) string {
	if p.isEmpty() {
		return tui.DimStyle.Render("No scratch paper")
	}

	var b strings.Builder

	if p.understanding != "" {
		b.WriteString(fmt.Sprintf("  %s\n", tui.ToolNameStyle.Render("Understanding")))
		b.WriteString("  " + p.understanding + "\n\n")
	}

	if len(p.plan) > 0 {
		b.WriteString(fmt.Sprintf("  %s\n", tui.ToolNameStyle.Render("Plan")))
		for i, step := range p.plan {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
		b.WriteByte('\n')
	}

	if len(p.risks) > 0 {
		b.WriteString(fmt.Sprintf("  %s\n", tui.ToolNameStyle.Render("Risks")))
		for _, risk := range p.risks {
			b.WriteString("  ⚠ " + risk + "\n")
		}
		b.WriteByte('\n')
	}

	if p.notes != "" {
		b.WriteString(fmt.Sprintf("  %s\n", tui.ToolNameStyle.Render("Notes")))
		b.WriteString("  " + p.notes + "\n")
	}

	return b.String()
}

// CellSight returns the scratch paper context for agent prompt injection.
func (p *ScratchPaperPanel) CellSight() tui.CellSight {
	fc := tui.CellSight{PanelID: p.ID()}

	if p.isEmpty() {
		return fc
	}

	fc.Kind = artifact.KindScratchPaper
	fc.CellTitle = "Scratch Paper"

	var fields []tui.SightField

	if p.understanding != "" {
		fields = append(fields, tui.SightField{
			Key:   "understanding",
			Value: p.understanding,
		})
	}

	if len(p.plan) > 0 {
		fields = append(fields, tui.SightField{
			Key:   "plan_steps",
			Value: fmt.Sprintf("%d", len(p.plan)),
		})
	}

	if len(p.risks) > 0 {
		fields = append(fields, tui.SightField{
			Key:   "risks",
			Value: fmt.Sprintf("%d", len(p.risks)),
		})
	}

	if p.notes != "" {
		fields = append(fields, tui.SightField{
			Key:   "notes",
			Value: p.notes,
		})
	}

	fc.Fields = fields
	return fc
}

// SightGate returns true — scratch paper is visible to agents by default.
func (p *ScratchPaperPanel) SightGate() bool { return true }

func (p *ScratchPaperPanel) isEmpty() bool {
	return p.understanding == "" && len(p.plan) == 0 && len(p.risks) == 0 && p.notes == ""
}

// Compile-time checks.
var (
	_ core.Panel  = (*ScratchPaperPanel)(nil)
	_ tui.Sighted = (*ScratchPaperPanel)(nil)
)
