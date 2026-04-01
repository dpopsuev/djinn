// plan_panel.go — Three-view plan visualization from ArtifactGraph (GOL-61, SPC-84).
//
// Three views: Overview (dependency diagram + progress), Goal Detail
// (segment list + status), Segment Detail (code + deps + ComponentMap).
// Composes existing render primitives (diagram, progress, tree, timeline).
package widgets

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/render"
	"github.com/dpopsuev/djinn/tui/core"

	tui "github.com/dpopsuev/djinn/tui"
)

const cursorPrefix = "▸ "

// PlanView is the current view mode of the PlanPanel.
type PlanView int

const (
	PlanViewOverview PlanView = iota
	PlanViewGoal
	PlanViewSegment
)

// PlanPanel renders an ArtifactGraph as three navigable views.
type PlanPanel struct {
	core.BasePanel
	graph  *artifact.Graph
	view   PlanView
	cursor int    // selected item in current view
	goalID string // selected goal (for detail views)
	segID  string // selected segment
	width  int
}

// NewPlanPanel creates a plan panel backed by an artifact graph.
func NewPlanPanel(graph *artifact.Graph) *PlanPanel {
	return &PlanPanel{
		BasePanel: core.NewBasePanel("plan", 0),
		graph:     graph,
	}
}

// Update handles keyboard navigation across three views.
func (p *PlanPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	if !p.Focused() {
		return p, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch km.String() {
	case "j", "down":
		p.cursor++
		p.clampCursor()
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "enter":
		p.dive()
	case "q", "esc":
		p.climb()
	case "tab":
		if p.view == PlanViewOverview {
			p.cursor++
			p.clampCursor()
		}
	}
	return p, nil
}

// View renders the current view.
func (p *PlanPanel) View(width int) string {
	p.width = width
	if p.graph == nil {
		return tui.DimStyle.Render("No plan loaded")
	}

	switch p.view {
	case PlanViewOverview:
		return p.renderOverview()
	case PlanViewGoal:
		return p.renderGoalDetail()
	case PlanViewSegment:
		return p.renderSegmentDetail()
	default:
		return tui.DimStyle.Render("Unknown view")
	}
}

// --- Overview ---

func (p *PlanPanel) renderOverview() string {
	all := p.graph.ListSorted()
	if len(all) == 0 {
		return tui.DimStyle.Render("Empty plan")
	}

	// Count progress.
	total := len(all)
	done := 0
	for i := range all {
		if all[i].Status == artifact.StatusComplete || all[i].Status == artifact.StatusDone {
			done++
		}
	}

	var b strings.Builder

	// Title + progress.
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	b.WriteString(fmt.Sprintf("  %d/%d segments  %d%%\n\n", done, total, pct))

	// Segment list with status glyphs.
	for i := range all {
		a := &all[i]
		prefix := "  "
		if i == p.cursor {
			prefix = cursorPrefix
		}

		glyph := statusGlyph(a.Status)
		title := a.Title
		if len(title) > 40 { //nolint:mnd // truncate for overview
			title = title[:37] + "..."
		}
		status := tui.DimStyle.Render(string(a.Status))

		b.WriteString(fmt.Sprintf("%s%s %-40s %s\n", prefix, glyph, title, status))
	}

	// Dependency diagram.
	dfg := render.ArtifactGraphToDataFlow(p.graph, "")
	if len(dfg.Edges) > 0 {
		b.WriteString("\n")
		depLine := renderDepChain(all)
		b.WriteString(depLine)
	}

	b.WriteString("\n")
	b.WriteString(tui.Hint("j/k navigate", "enter drill", "q back"))

	return b.String()
}

// --- Goal Detail ---

func (p *PlanPanel) renderGoalDetail() string {
	a, err := p.graph.Get(p.goalID)
	if err != nil {
		return tui.DimStyle.Render("Goal not found: " + p.goalID)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s\n", statusGlyph(a.Status), a.Title))
	b.WriteString(fmt.Sprintf("  Status: %s  Owner: %s\n\n", a.Status, orDash(a.Owner)))

	// Children (if any).
	children := p.childArtifacts(p.goalID)
	if len(children) > 0 {
		for i := range children {
			c := &children[i]
			prefix := "  "
			if i == p.cursor {
				prefix = "▸ "
			}
			files := len(c.Components.Files) + len(c.Components.Directories)
			fileStr := ""
			if files > 0 {
				fileStr = tui.DimStyle.Render(fmt.Sprintf(" (%d files)", files))
			}
			b.WriteString(fmt.Sprintf("%s%s %-6s %s%s\n", prefix, statusGlyph(c.Status), c.ID, c.Title, fileStr))
		}
	}

	// Dependencies.
	if len(a.DependsOn) > 0 {
		b.WriteString("\n  ▲ depends: ")
		b.WriteString(tui.DimStyle.Render(strings.Join(a.DependsOn, ", ")))
		b.WriteByte('\n')
	}

	// Content.
	if a.Content != "" {
		b.WriteString("\n")
		b.WriteString(tui.DimStyle.Render("  " + truncate(a.Content, 200))) //nolint:mnd // content preview
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	b.WriteString(tui.Hint("j/k navigate", "enter drill", "q overview"))

	return b.String()
}

// --- Segment Detail ---

func (p *PlanPanel) renderSegmentDetail() string {
	a, err := p.graph.Get(p.segID)
	if err != nil {
		return tui.DimStyle.Render("Segment not found: " + p.segID)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s\n", statusGlyph(a.Status), a.Title))
	b.WriteString(fmt.Sprintf("  ID: %s  Status: %s  Owner: %s\n\n", a.ID, a.Status, orDash(a.Owner)))

	// ComponentMap as tree.
	if len(a.Components.Files) > 0 || len(a.Components.Directories) > 0 {
		treeData := buildComponentTree(a)
		msg := tui.RenderPanelMsg{Type: tui.RenderTypeTree, Title: "Components", Data: treeData}
		b.WriteString(tui.RenderPanel(msg, p.width))
		b.WriteByte('\n')
	}

	// Content.
	if a.Content != "" {
		b.WriteString("  Content:\n")
		for _, line := range strings.Split(a.Content, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}

	// Sections.
	for name, text := range a.Sections {
		if name == "content" {
			continue // already shown
		}
		b.WriteString(fmt.Sprintf("\n  %s:\n", tui.ToolNameStyle.Render(name)))
		b.WriteString("  " + truncate(text, 300) + "\n") //nolint:mnd // section preview
	}

	// Dependencies.
	if len(a.DependsOn) > 0 {
		b.WriteString("\n  ▲ depends: " + tui.DimStyle.Render(strings.Join(a.DependsOn, ", ")) + "\n")
	}

	// Annotations.
	if len(a.Annotations) > 0 {
		b.WriteString("\n  Annotations:\n")
		for i := range a.Annotations {
			b.WriteString(fmt.Sprintf("    %s %s\n", a.Annotations[i].Kind, a.Annotations[i].Comment))
		}
	}

	b.WriteString("\n")
	b.WriteString(tui.Hint("q back", ":claim", ":+ :- :~ annotate"))

	return b.String()
}

// --- Navigation ---

func (p *PlanPanel) dive() {
	switch p.view {
	case PlanViewOverview:
		all := p.graph.ListSorted()
		if p.cursor < len(all) {
			p.goalID = all[p.cursor].ID
			p.view = PlanViewGoal
			p.cursor = 0
		}
	case PlanViewGoal:
		children := p.childArtifacts(p.goalID)
		if p.cursor < len(children) {
			p.segID = children[p.cursor].ID
			p.view = PlanViewSegment
			p.cursor = 0
		}
	case PlanViewSegment:
		// Already at deepest level.
	}
}

func (p *PlanPanel) climb() {
	switch p.view {
	case PlanViewSegment:
		p.view = PlanViewGoal
		p.cursor = 0
	case PlanViewGoal:
		p.view = PlanViewOverview
		p.cursor = 0
	case PlanViewOverview:
		// Already at top.
	}
}

func (p *PlanPanel) clampCursor() {
	count := p.itemCount()
	if count > 0 && p.cursor >= count {
		p.cursor = count - 1
	}
}

func (p *PlanPanel) itemCount() int {
	switch p.view {
	case PlanViewOverview:
		return len(p.graph.ListSorted())
	case PlanViewGoal:
		return len(p.childArtifacts(p.goalID))
	default:
		return 0
	}
}

func (p *PlanPanel) childArtifacts(parentID string) []artifact.Artifact {
	all := p.graph.ListSorted()
	var children []artifact.Artifact
	for i := range all {
		if all[i].Parent == parentID {
			children = append(children, all[i])
		}
	}
	return children
}

// FocusContext returns the currently focused artifact's context.
func (p *PlanPanel) FocusContext() tui.FocusContext {
	fc := tui.FocusContext{PanelID: p.ID()}

	switch p.view {
	case PlanViewOverview:
		all := p.graph.ListSorted()
		if p.cursor < len(all) {
			fc.ElementID = all[p.cursor].ID
			fc.ElementTitle = all[p.cursor].Title
			fc.Kind = all[p.cursor].Kind
			fc.Metadata = map[string]string{"status": string(all[p.cursor].Status), "view": "overview"}
		}
	case PlanViewGoal:
		fc.ElementID = p.goalID
		if a, err := p.graph.Get(p.goalID); err == nil {
			fc.ElementTitle = a.Title
			fc.Kind = a.Kind
			fc.Metadata = map[string]string{"status": string(a.Status), "view": "goal"}
		}
	case PlanViewSegment:
		fc.ElementID = p.segID
		if a, err := p.graph.Get(p.segID); err == nil {
			fc.ElementTitle = a.Title
			fc.Kind = a.Kind
			fc.Metadata = map[string]string{"status": string(a.Status), "view": "segment"}
		}
	}
	return fc
}

// Compile-time checks.
var (
	_ core.Panel               = (*PlanPanel)(nil)
	_ tui.FocusContextProvider = (*PlanPanel)(nil)
)

// --- Helpers ---

func statusGlyph(s artifact.Status) string {
	switch s {
	case artifact.StatusComplete, artifact.StatusDone:
		return tui.ToolSuccessStyle.Render("⬢")
	case artifact.StatusInProgress, artifact.StatusActive:
		return tui.ToolNameStyle.Render("⬡")
	case artifact.StatusClaimed:
		return tui.ToolNameStyle.Render("◆")
	case artifact.StatusInvalidated:
		return tui.ErrorStyle.Render("●")
	default:
		return tui.DimStyle.Render("○")
	}
}

func renderDepChain(all []artifact.Artifact) string {
	parts := make([]string, 0, len(all))
	for i := range all {
		parts = append(parts, all[i].ID)
	}
	return "  " + tui.DimStyle.Render(strings.Join(parts, " ──→ ")) + "\n"
}

func buildComponentTree(a *artifact.Artifact) string {
	root := map[string]any{
		"label":    "Components",
		"children": []map[string]string{},
	}

	children := make([]map[string]string, 0, len(a.Components.Directories)+len(a.Components.Files))
	for _, d := range a.Components.Directories {
		children = append(children, map[string]string{"label": "📁 " + d})
	}
	for _, f := range a.Components.Files {
		children = append(children, map[string]string{"label": "📄 " + f})
	}
	for _, s := range a.Components.Symbols {
		children = append(children, map[string]string{"label": "λ " + s})
	}
	root["children"] = children

	data, _ := json.Marshal(map[string]any{"root": root}) //nolint:errcheck // internal construction
	return string(data)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit-3] + "..."
}
