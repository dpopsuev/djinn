// turn_envelope.go — TurnEnvelope wraps a conversation turn in a bordered box.
// Renders user input + tool calls + agent response + thinking as a single
// collapsible bordered unit. Children() exposes tool EnvelopePanels for Dive.
// No domain imports — pure TUI component.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// turnBorder is the dim rounded border used for turn envelopes.
var turnBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#808080", Dark: "#505050"})

// TurnEnvelope wraps a full conversation turn (user + agent + tools) in a
// bordered box with a "Turn #N" title. Collapsible to a single-line summary.
type TurnEnvelope struct {
	BasePanel
	turnIdx   int
	userInput string
	response  strings.Builder
	tools     []*EnvelopePanel
	thinking  string
	complete  bool
}

var _ Panel = (*TurnEnvelope)(nil)

// NewTurnEnvelope creates a turn envelope for the given turn index and user input.
func NewTurnEnvelope(turnIdx int, userInput string) *TurnEnvelope {
	return &TurnEnvelope{
		BasePanel: NewBasePanel(fmt.Sprintf("turn-%d", turnIdx), 0),
		turnIdx:   turnIdx,
		userInput: userInput,
	}
}

// AddText appends text to the agent response.
func (t *TurnEnvelope) AddText(text string) {
	t.response.WriteString(text)
}

// AddThinking sets the thinking/reasoning block.
func (t *TurnEnvelope) AddThinking(text string) {
	t.thinking = text
}

// AddTool appends a tool envelope to this turn.
func (t *TurnEnvelope) AddTool(env *EnvelopePanel) {
	t.tools = append(t.tools, env)
}

// Complete marks this turn as finished.
func (t *TurnEnvelope) Complete() {
	t.complete = true
}

func (t *TurnEnvelope) Collapsible() bool { return true }

// Children returns tool envelopes as Panels for Dive navigation.
func (t *TurnEnvelope) Children() []Panel {
	panels := make([]Panel, len(t.tools))
	for i, tool := range t.tools {
		panels[i] = tool
	}
	return panels
}

func (t *TurnEnvelope) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return t, nil
}

// View renders the turn envelope. Collapsed = single-line summary.
// Expanded = bordered box with title, user input, tools, response, thinking.
func (t *TurnEnvelope) View(width int) string {
	if t.collapsed {
		return t.collapsedView()
	}
	return t.expandedView(width)
}

func (t *TurnEnvelope) collapsedView() string {
	summary := t.userInput
	if len(summary) > 50 {
		summary = summary[:47] + "..."
	}
	return DimStyle.Render(fmt.Sprintf("Turn #%d: %s", t.turnIdx, summary))
}

func (t *TurnEnvelope) expandedView(width int) string {
	var sb strings.Builder

	// User input with > prefix.
	sb.WriteString(UserStyle.Render(LabelUser) + t.userInput)

	// Tool summaries.
	for _, tool := range t.tools {
		sb.WriteByte('\n')
		sb.WriteString(tool.View(width - 4)) // account for border padding
	}

	// Agent response text.
	resp := t.response.String()
	if resp != "" {
		sb.WriteByte('\n')
		sb.WriteString(resp)
	}

	// Thinking block (dim, only if non-empty).
	if t.thinking != "" {
		sb.WriteByte('\n')
		sb.WriteString(DimStyle.Render(SpinnerFrames[0] + " " + t.thinking))
	}

	// Render bordered box with title.
	title := fmt.Sprintf(" Turn #%d ", t.turnIdx)
	innerWidth := width - 2 // border chars
	if innerWidth < 1 {
		innerWidth = 1
	}

	return turnBorder.
		Width(innerWidth).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Render(title + "\n" + sb.String())
}
