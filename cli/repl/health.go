// health.go — HealthReporter interface and status line rendering.
package repl

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HealthStatus represents component health.
type HealthStatus int

const (
	StatusGreen  HealthStatus = iota // healthy
	StatusYellow                     // degraded
	StatusRed                        // fatal
)

// HealthReport is a single component's health.
type HealthReport struct {
	Component string
	Status    HealthStatus
	Message   string // optional detail
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

// renderStatusLine builds the unified status line.
// Left: workspace │ driver/model │ mode
// Center: tokens │ turns
// Right: health indicators (hidden when all green)
func renderStatusLine(workspace, driver, model, mode string, tokensIn, tokensOut, turns int, reports []HealthReport) string {
	var parts []string

	// Left: identity
	wsName := workspace
	if wsName == "" {
		wsName = "(ephemeral)"
	}
	driverModel := model
	if driver != "" && driver != "claude" {
		driverModel = driver + "/" + model
	}
	parts = append(parts, fmt.Sprintf("  %s │ %s │ %s", wsName, driverModel, mode))

	// Center: metrics
	parts = append(parts, fmt.Sprintf("%d in, %d out │ %d turns", tokensIn, tokensOut, turns))

	// Right: health
	healthStr := renderHealth(reports)
	if healthStr != "" {
		parts = append(parts, healthStr)
	}

	return statusStyle.Render(strings.Join(parts, " │ "))
}

func renderHealth(reports []HealthReport) string {
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

	// Show non-green + green count
	if greenCount > 0 {
		indicators = append(indicators, healthGreen.Render(fmt.Sprintf("✓ %d", greenCount)))
	}
	return strings.Join(indicators, " ")
}
