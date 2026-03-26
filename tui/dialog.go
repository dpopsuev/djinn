// dialog.go -- DialogPanel is a modal dialog with selectable action buttons.
// Renders a bordered box with title, message, and horizontally arranged actions.
// Tab cycles actions. Enter emits DialogResultMsg. Esc cancels.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DialogAction is a selectable button in a DialogPanel.
type DialogAction struct {
	ID    string // "allow", "deny", "cancel"
	Label string // "[Allow]", "[Deny]"
}

// DialogPanel is a modal dialog with a title, message, and action buttons.
type DialogPanel struct {
	BasePanel
	title     string
	message   string
	actions   []DialogAction
	actionIdx int
}

var _ Panel = (*DialogPanel)(nil)

// NewDialogPanel creates a dialog with the given title, message, and actions.
func NewDialogPanel(title, message string, actions ...DialogAction) *DialogPanel {
	return &DialogPanel{
		BasePanel: NewBasePanel("dialog", 0),
		title:     title,
		message:   message,
		actions:   actions,
	}
}

// SelectedAction returns the ID of the currently selected action.
func (p *DialogPanel) SelectedAction() string {
	if len(p.actions) == 0 {
		return ""
	}
	return p.actions[p.actionIdx].ID
}

// Update handles key input: Tab cycles actions, Enter confirms, Esc cancels.
func (p *DialogPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch msg.Type {
		case tea.KeyTab:
			if len(p.actions) > 0 {
				p.actionIdx = (p.actionIdx + 1) % len(p.actions)
			}
		case tea.KeyEnter:
			return p, func() tea.Msg {
				return DialogResultMsg{ActionID: p.SelectedAction()}
			}
		case tea.KeyEsc:
			return p, func() tea.Msg {
				return DialogResultMsg{ActionID: "cancel"}
			}
		}
	}
	return p, nil
}

// dialogBorder is the rounded border style for dialogs.
var dialogBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(RedHatRed)

// selectedActionStyle highlights the selected action button.
var selectedActionStyle = lipgloss.NewStyle().Bold(true).Reverse(true)

// normalActionStyle renders unselected action buttons.
var normalActionStyle = lipgloss.NewStyle().Faint(true)

// View renders the dialog as a bordered box with title, message, and actions.
func (p *DialogPanel) View(width int) string {
	innerWidth := width - 2
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Title line.
	titleLine := lipgloss.NewStyle().Bold(true).Render(p.title)

	// Message centered.
	msgStyle := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)
	messageLine := msgStyle.Render(p.message)

	// Action buttons.
	var buttons []string
	for i, a := range p.actions {
		if i == p.actionIdx {
			buttons = append(buttons, selectedActionStyle.Render(a.Label))
		} else {
			buttons = append(buttons, normalActionStyle.Render(a.Label))
		}
	}
	actionsLine := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(strings.Join(buttons, "  "))

	// Compose inner content.
	content := strings.Join([]string{titleLine, "", messageLine, "", actionsLine}, "\n")

	return dialogBorder.Width(innerWidth).Render(content)
}
