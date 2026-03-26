// render_panel.go — renders structured panels from RenderPanelMsg.
//
// Each panel type has a dedicated renderer that produces a bordered,
// styled lipgloss string. Called inline in the conversation when the
// agent uses the render tool.
package tui

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// RenderPanel renders a RenderPanelMsg as a styled string.
func RenderPanel(msg RenderPanelMsg, width int) string {
	if width < 20 {
		width = 20
	}
	inner := width - 4 // border + padding

	var content string
	switch msg.Type {
	case "table":
		content = renderTable(msg.Data, inner)
	case "tree":
		content = renderTree(msg.Data, inner)
	case "progress":
		content = renderProgress(msg.Data, inner)
	case "chart":
		content = renderChart(msg.Data, inner)
	default:
		content = DimStyle.Render("[unknown render type: " + msg.Type + "]")
	}

	// Build bordered panel.
	title := " " + msg.Title + " "
	topLen := inner - len(title)
	if topLen < 2 {
		topLen = 2
	}
	leftPad := topLen / 2
	rightPad := topLen - leftPad

	var sb strings.Builder
	sb.WriteString(DimStyle.Render("╭" + strings.Repeat("─", leftPad) + title + strings.Repeat("─", rightPad) + "╮"))
	sb.WriteByte('\n')

	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}
		sb.WriteString(DimStyle.Render("│ "))
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	sb.WriteString(DimStyle.Render("╰" + strings.Repeat("─", inner) + "╯"))
	return sb.String()
}

// --- Table renderer ---

type tableData struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

func renderTable(data string, width int) string {
	var td tableData
	if err := json.Unmarshal([]byte(data), &td); err != nil {
		return DimStyle.Render("[invalid table data]")
	}
	if len(td.Columns) == 0 {
		return DimStyle.Render("[empty table]")
	}

	// Compute column widths.
	colWidths := make([]int, len(td.Columns))
	for i, col := range td.Columns {
		colWidths[i] = len(col)
	}
	for _, row := range td.Rows {
		for i, cell := range row {
			if i >= len(colWidths) {
				break
			}
			s := fmt.Sprintf("%v", cell)
			if len(s) > colWidths[i] {
				colWidths[i] = len(s)
			}
		}
	}

	var sb strings.Builder

	// Header.
	for i, col := range td.Columns {
		if i > 0 {
			sb.WriteString(DimStyle.Render(" │ "))
		}
		sb.WriteString(ToolNameStyle.Render(pad(col, colWidths[i])))
	}
	sb.WriteByte('\n')

	// Separator.
	for i, w := range colWidths {
		if i > 0 {
			sb.WriteString(DimStyle.Render("─┼─"))
		}
		sb.WriteString(DimStyle.Render(strings.Repeat("─", w)))
	}
	sb.WriteByte('\n')

	// Rows.
	for _, row := range td.Rows {
		for i, cell := range row {
			if i >= len(colWidths) {
				break
			}
			if i > 0 {
				sb.WriteString(DimStyle.Render(" │ "))
			}
			sb.WriteString(pad(fmt.Sprintf("%v", cell), colWidths[i]))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// --- Tree renderer ---

type treeData struct {
	Root treeNode `json:"root"`
}

type treeNode struct {
	Label    string     `json:"label"`
	Children []treeNode `json:"children,omitempty"`
}

func renderTree(data string, _ int) string {
	var td treeData
	if err := json.Unmarshal([]byte(data), &td); err != nil {
		return DimStyle.Render("[invalid tree data]")
	}

	var sb strings.Builder
	sb.WriteString(ToolNameStyle.Render(td.Root.Label))
	sb.WriteByte('\n')
	renderTreeChildren(&sb, td.Root.Children, "")
	return sb.String()
}

func renderTreeChildren(sb *strings.Builder, children []treeNode, prefix string) {
	for i, child := range children {
		isLast := i == len(children)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}
		sb.WriteString(DimStyle.Render(prefix + connector))
		sb.WriteString(child.Label)
		sb.WriteByte('\n')
		if len(child.Children) > 0 {
			renderTreeChildren(sb, child.Children, prefix+childPrefix)
		}
	}
}

// --- Progress renderer ---

type progressData struct {
	Done  int      `json:"done"`
	Total int      `json:"total"`
	Items []string `json:"items,omitempty"`
}

func renderProgress(data string, width int) string {
	var pd progressData
	if err := json.Unmarshal([]byte(data), &pd); err != nil {
		return DimStyle.Render("[invalid progress data]")
	}
	if pd.Total <= 0 {
		pd.Total = 1
	}

	pct := float64(pd.Done) / float64(pd.Total) * 100
	barWidth := width - 12 // room for "███░░ 100%"
	if barWidth < 10 {
		barWidth = 10
	}
	filled := int(math.Round(float64(barWidth) * float64(pd.Done) / float64(pd.Total)))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	var sb strings.Builder

	// Progress bar.
	bar := ToolSuccessStyle.Render(strings.Repeat("█", filled)) +
		DimStyle.Render(strings.Repeat("░", empty))
	sb.WriteString(bar)
	sb.WriteString(fmt.Sprintf(" %3.0f%%", pct))
	sb.WriteString(DimStyle.Render(fmt.Sprintf(" (%d/%d)", pd.Done, pd.Total)))
	sb.WriteByte('\n')

	// Optional checklist.
	for i, item := range pd.Items {
		if i < pd.Done {
			sb.WriteString(ToolSuccessStyle.Render("  ☑ "))
		} else {
			sb.WriteString(DimStyle.Render("  ☐ "))
		}
		sb.WriteString(item)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// --- Chart renderer (sparkline) ---

type chartData struct {
	Kind   string    `json:"kind"` // sparkline, bar
	Values []float64 `json:"values"`
	Labels []string  `json:"labels,omitempty"`
}

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func renderChart(data string, width int) string {
	var cd chartData
	if err := json.Unmarshal([]byte(data), &cd); err != nil {
		return DimStyle.Render("[invalid chart data]")
	}
	if len(cd.Values) == 0 {
		return DimStyle.Render("[no data]")
	}

	switch cd.Kind {
	case "bar":
		return renderBarChart(cd, width)
	default: // sparkline
		return renderSparkline(cd)
	}
}

func renderSparkline(cd chartData) string {
	min, max := cd.Values[0], cd.Values[0]
	for _, v := range cd.Values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	rng := max - min
	if rng == 0 {
		rng = 1
	}

	var sb strings.Builder
	for _, v := range cd.Values {
		idx := int((v - min) / rng * 7)
		if idx > 7 {
			idx = 7
		}
		sb.WriteRune(sparkBlocks[idx])
	}

	// Labels on next line if present.
	if len(cd.Labels) > 0 {
		sb.WriteByte('\n')
		sb.WriteString(DimStyle.Render(strings.Join(cd.Labels, " ")))
	}

	return sb.String()
}

func renderBarChart(cd chartData, width int) string {
	max := cd.Values[0]
	for _, v := range cd.Values {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		max = 1
	}

	// Find max label width.
	labelWidth := 0
	for i := range cd.Values {
		label := fmt.Sprintf("%d", i)
		if i < len(cd.Labels) {
			label = cd.Labels[i]
		}
		if len(label) > labelWidth {
			labelWidth = len(label)
		}
	}

	barMax := width - labelWidth - 10 // room for label + value
	if barMax < 5 {
		barMax = 5
	}

	var sb strings.Builder
	for i, v := range cd.Values {
		label := fmt.Sprintf("%d", i)
		if i < len(cd.Labels) {
			label = cd.Labels[i]
		}
		barLen := int(v / max * float64(barMax))
		sb.WriteString(pad(label, labelWidth))
		sb.WriteString(" ")
		sb.WriteString(ToolSuccessStyle.Render(strings.Repeat("█", barLen)))
		sb.WriteString(fmt.Sprintf(" %.1f", v))
		sb.WriteByte('\n')
	}

	return sb.String()
}
