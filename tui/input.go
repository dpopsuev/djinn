// input.go — InputPanel wraps textarea with focus indicator and history.
package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// InputPanel is the user input area.
type InputPanel struct {
	BasePanel
	textarea textarea.Model
	history  []string
	histIdx  int
	onSubmit func(string) // callback when Enter pressed
}

// NewInputPanel creates the input panel.
func NewInputPanel() *InputPanel {
	ta := textarea.New()
	ta.Prompt = UserStyle.Render(LabelUser)
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.CharLimit = 0
	ta.Focus()

	return &InputPanel{
		BasePanel: NewBasePanel("input", 1),
		textarea:  ta,
		histIdx:   -1,
	}
}

// OnSubmit sets the callback for Enter key.
func (p *InputPanel) OnSubmit(fn func(string)) {
	p.onSubmit = fn
}

// Value returns the current input text.
func (p *InputPanel) Value() string {
	return p.textarea.Value()
}

// SetValue sets the input text.
func (p *InputPanel) SetValue(s string) {
	p.textarea.SetValue(s)
}

// Reset clears the input.
func (p *InputPanel) Reset() {
	p.textarea.Reset()
}

// Focus activates the textarea cursor.
func (p *InputPanel) FocusInput() {
	p.textarea.Focus()
}

// Blur deactivates the textarea cursor.
func (p *InputPanel) BlurInput() {
	p.textarea.Blur()
}

// AddHistory records a submitted prompt.
func (p *InputPanel) AddHistory(s string) {
	p.history = append(p.history, s)
	p.histIdx = -1
}

// HistoryUp recalls the previous prompt.
func (p *InputPanel) HistoryUp() {
	if len(p.history) == 0 {
		return
	}
	if p.histIdx == -1 {
		p.histIdx = len(p.history) - 1
	} else if p.histIdx > 0 {
		p.histIdx--
	}
	p.textarea.SetValue(p.history[p.histIdx])
}

// HistoryDown recalls the next prompt.
func (p *InputPanel) HistoryDown() {
	if p.histIdx < 0 {
		return
	}
	p.histIdx++
	if p.histIdx >= len(p.history) {
		p.histIdx = -1
		p.textarea.SetValue("")
	} else {
		p.textarea.SetValue(p.history[p.histIdx])
	}
}

func (p *InputPanel) SetFocus(f bool) {
	p.focused = f
	if f {
		p.textarea.Focus()
	} else {
		p.textarea.Blur()
	}
}

func (p *InputPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	if !p.focused {
		return p, nil
	}
	var cmd tea.Cmd
	p.textarea, cmd = p.textarea.Update(msg)
	return p, cmd
}

func (p *InputPanel) View(width int) string {
	return p.textarea.View()
}
