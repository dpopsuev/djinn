// onboarding.go — First-run interactive driver selection (GOL-63, SPC-86).
//
// OnboardingModel is a standalone Bubbletea model that runs BEFORE the REPL.
// It shows detected CLIs as a radio list, or a path input if none found.
// Selection emits the chosen driver — the caller writes config and starts the REPL.
package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	tui "github.com/dpopsuev/djinn/tui"
)

// DetectedCLI represents an agent CLI found on PATH (passed from app/arsenal.go).
type DetectedCLI struct {
	Name    string // "cursor", "claude", "gemini", "codex", "ollama"
	Binary  string // path to binary
	Version string // version string
	Source  string // "cli" or "api-key"
}

// OnboardingResult is returned when the user makes a selection.
type OnboardingResult struct {
	Selected *DetectedCLI // nil if user quit
	Quit     bool
}

// OnboardingModel is the Bubbletea model for first-run driver selection.
type OnboardingModel struct {
	drivers []DetectedCLI
	cursor  int
	width   int
	height  int
	result  OnboardingResult
	done    bool
}

// NewOnboardingModel creates the onboarding model with detected CLIs.
func NewOnboardingModel(drivers []DetectedCLI) OnboardingModel {
	return OnboardingModel{
		drivers: drivers,
	}
}

// Result returns the selection after the program exits.
func (m OnboardingModel) Result() OnboardingResult {
	return m.result
}

// Init implements tea.Model.
func (m OnboardingModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.drivers)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(m.drivers) > 0 && m.cursor < len(m.drivers) {
				m.result = OnboardingResult{Selected: &m.drivers[m.cursor]}
				m.done = true
				return m, tea.Quit
			}
		case "q", "esc", "ctrl+c":
			m.result = OnboardingResult{Quit: true}
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m OnboardingModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Logo.
	logo := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#EE0000"}).Bold(true)
	b.WriteString(logo.Render(tui.DjinnLogo))
	b.WriteString("\n\n")

	if len(m.drivers) == 0 {
		b.WriteString(m.renderNoDrivers())
	} else {
		b.WriteString(m.renderDriverList())
	}

	return b.String()
}

func (m OnboardingModel) renderDriverList() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true)
	b.WriteString(title.Render("  Select your agent backend:"))
	b.WriteString("\n\n")

	for i, d := range m.drivers {
		prefix := "    "
		if i == m.cursor {
			prefix = "  ● "
		}

		name := lipgloss.NewStyle().Bold(true).Render(padDriverName(d.Name))
		path := tui.DimStyle.Render(d.Binary)
		version := ""
		if d.Version != "" {
			// Take first line only, truncate.
			v := strings.SplitN(d.Version, "\n", 2)[0] //nolint:mnd // first line
			if len(v) > 30 {                           //nolint:mnd // truncate long versions
				v = v[:27] + "..."
			}
			version = tui.DimStyle.Render("  " + v)
		}
		if d.Source == "api-key" {
			path = tui.DimStyle.Render("(API key set)")
		}

		b.WriteString(fmt.Sprintf("%s%s  %s%s\n", prefix, name, path, version))
	}

	b.WriteString("\n")
	b.WriteString(tui.DimStyle.Render("  j/k navigate · enter select · q quit"))
	b.WriteString("\n")

	return b.String()
}

func (m OnboardingModel) renderNoDrivers() string {
	var b strings.Builder

	b.WriteString("  No agent CLI found on PATH.\n\n")
	b.WriteString("  Install one of:\n\n")

	installs := []struct {
		name string
		url  string
	}{
		{"cursor", "https://cursor.com/downloads"},
		{"claude", "https://docs.anthropic.com/claude-code"},
		{"gemini", "https://cloud.google.com/gemini"},
		{"codex", "https://openai.com/codex"},
		{"ollama", "https://ollama.com"},
	}

	for _, inst := range installs {
		name := lipgloss.NewStyle().Bold(true).Render(padDriverName(inst.name))
		url := tui.DimStyle.Render(inst.url)
		b.WriteString(fmt.Sprintf("    %s  %s\n", name, url))
	}

	b.WriteString("\n")
	b.WriteString("  Or set ANTHROPIC_API_KEY for direct Claude API access.\n\n")
	b.WriteString(tui.DimStyle.Render("  q quit"))
	b.WriteString("\n")

	return b.String()
}

func padDriverName(name string) string {
	const width = 10
	if len(name) >= width {
		return name
	}
	return name + strings.Repeat(" ", width-len(name))
}

// RunOnboarding runs the onboarding TUI and returns the selected driver.
// This is a blocking call — it runs its own tea.Program.
func RunOnboarding(drivers []DetectedCLI) (OnboardingResult, error) {
	m := NewOnboardingModel(drivers)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return OnboardingResult{Quit: true}, err
	}

	return finalModel.(OnboardingModel).Result(), nil
}
