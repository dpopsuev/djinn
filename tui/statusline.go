package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HealthStatus represents component health.
type HealthStatus int

const (
	StatusGreen   HealthStatus = iota
	StatusYellow
	StatusRed
	StatusOffline // server declared but not running — dimmed, not warning
)

// HealthReport is a single component's health.
type HealthReport struct {
	Component string
	Status    HealthStatus
	Message   string
}

// HealthReporter is implemented by components that report health.
type HealthReporter interface {
	Health() HealthReport
}

// Status line field keys.
const (
	fieldWorkspace = "scope"
	fieldModel     = "model"
	fieldMode      = "mode"
	fieldTokens    = "tok"
	fieldTurns     = "turns"
	fieldMCP       = "mcp"
)

// Status line display constants.
const (
	labelEphemeral = "general"
	driverClaude   = "claude"
)

// Status line styles.
var (
	healthGreen  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})
	healthYellow = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"})
	healthRed    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"})

	fieldKeyStyle   = lipgloss.NewStyle().Faint(true)
	fieldValueStyle = lipgloss.NewStyle().Bold(true)
	fieldSepStyle   = lipgloss.NewStyle().Faint(true)
)

// StatusField is a single key:value field in the status line.
type StatusField struct {
	Key   string
	Value string
	Style lipgloss.Style // optional override for value rendering
}

// FormatField renders a single key:value field with styled key and value.
func FormatField(f StatusField) string {
	val := fieldValueStyle.Render(f.Value)
	if f.Style.Value() != "" {
		val = f.Style.Render(f.Value)
	}
	return fieldKeyStyle.Render(f.Key+":") + val
}

// FormatStatusLine renders a list of fields separated by styled │.
func FormatStatusLine(fields []StatusField) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, FormatField(f))
	}
	sep := fieldSepStyle.Render(" │ ")
	return "  " + strings.Join(parts, sep)
}

// RenderStatusLine builds the unified status line from structured data.
func RenderStatusLine(workspace, driverName, model, mode string, tokensIn, tokensOut, turns int, reports []HealthReport) string {
	wsName := workspace
	if wsName == "" {
		wsName = labelEphemeral
	}
	driverModel := model
	if driverName != "" && driverName != driverClaude {
		driverModel = driverName + "/" + model
	}

	fields := make([]StatusField, 0, 6)
	fields = append(fields,
		StatusField{Key: fieldWorkspace, Value: wsName},
		StatusField{Key: fieldModel, Value: driverModel},
		StatusField{Key: fieldMode, Value: mode},
		StatusField{Key: fieldTokens, Value: fmt.Sprintf("%d/%d", tokensIn, tokensOut)},
		StatusField{Key: fieldTurns, Value: fmt.Sprintf("%d", turns)},
	)

	healthStr := RenderHealth(reports)
	if healthStr != "" {
		fields = append(fields, StatusField{Key: fieldMCP, Value: healthStr})
	}

	return StatusStyle.Render(FormatStatusLine(fields))
}

// RenderHealth renders health indicators with color coding.
// Shows individual component names so you know exactly what's up/down.
func RenderHealth(reports []HealthReport) string {
	if len(reports) == 0 {
		return ""
	}

	indicators := make([]string, 0, len(reports))
	for _, r := range reports {
		switch r.Status {
		case StatusGreen:
			indicators = append(indicators, healthGreen.Render("✓"+r.Component))
		case StatusYellow:
			indicators = append(indicators, healthYellow.Render("⚠"+r.Component))
		case StatusRed:
			indicators = append(indicators, healthRed.Render("✗"+r.Component))
		case StatusOffline:
			indicators = append(indicators, fieldKeyStyle.Render("·"+r.Component))
		}
	}
	return strings.Join(indicators, " ")
}

// RenderHealthCompact renders a compact health summary (count only).
// Used for small terminal breakpoints.
func RenderHealthCompact(reports []HealthReport) string {
	if len(reports) == 0 {
		return ""
	}
	green, total := 0, len(reports)
	for _, r := range reports {
		if r.Status == StatusGreen {
			green++
		}
	}
	if green == total {
		return healthGreen.Render(fmt.Sprintf("✓%d", green))
	}
	return fmt.Sprintf("%d/%d", green, total)
}
