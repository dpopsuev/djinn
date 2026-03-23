package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HealthStatus represents component health.
type HealthStatus int

const (
	StatusGreen  HealthStatus = iota
	StatusYellow
	StatusRed
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

// Status line styles.
var (
	healthGreen  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})
	healthYellow = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"})
	healthRed    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"})
)

// RenderStatusLine builds the unified status line.
func RenderStatusLine(workspace, driverName, model, mode string, tokensIn, tokensOut, turns int, reports []HealthReport) string {
	var parts []string

	wsName := workspace
	if wsName == "" {
		wsName = "(ephemeral)"
	}
	driverModel := model
	if driverName != "" && driverName != "claude" {
		driverModel = driverName + "/" + model
	}
	parts = append(parts, fmt.Sprintf("  ws:%s │ model:%s │ mode:%s", wsName, driverModel, mode))
	parts = append(parts, fmt.Sprintf("tok:%d/%d │ turns:%d", tokensIn, tokensOut, turns))

	healthStr := RenderHealth(reports)
	if healthStr != "" {
		parts = append(parts, healthStr)
	}

	return StatusStyle.Render(strings.Join(parts, " │ "))
}

// RenderHealth renders health indicators.
func RenderHealth(reports []HealthReport) string {
	if len(reports) == 0 {
		return ""
	}

	allGreen := true
	greenCount := 0
	var indicators []string

	for _, r := range reports {
		switch r.Status {
		case StatusGreen:
			greenCount++
		case StatusYellow:
			allGreen = false
			indicators = append(indicators, healthYellow.Render("⚠ "+r.Component))
		case StatusRed:
			allGreen = false
			indicators = append(indicators, healthRed.Render("✗ "+r.Component))
		}
	}

	if allGreen {
		if greenCount > 0 {
			return healthGreen.Render(fmt.Sprintf("✓ %d mcp", greenCount))
		}
		return ""
	}

	if greenCount > 0 {
		indicators = append(indicators, healthGreen.Render(fmt.Sprintf("✓ %d", greenCount)))
	}
	return strings.Join(indicators, " ")
}
