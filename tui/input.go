// input.go — InputPanel wraps textarea with focus indicator and history.
package tui

import (
	"strings"

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
	visible  bool

	// Tab completion state.
	completions    []string // sorted command names
	matches        []string // current matches for prefix
	compPrefix     string   // the prefix being completed
	compIdx        int      // index into matches
	lastCompletion string   // last completed value (for cycle detection)
}

const panelIDInput = "input"

var _ Panel = (*InputPanel)(nil)

// NewInputPanel creates the input panel.
func NewInputPanel() *InputPanel {
	ta := textarea.New()
	ta.Prompt = UserStyle.Render(LabelUser)
	ta.Placeholder = `Try "explain this codebase"`
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.CharLimit = 0
	// Style user input text in green (BUG-30).
	ta.FocusedStyle.Text = UserStyle
	ta.BlurredStyle.Text = DimStyle
	ta.Focus()

	return &InputPanel{
		BasePanel: NewBasePanel(panelIDInput, 1),
		textarea:  ta,
		histIdx:   -1,
		visible:   true,
		compIdx:   -1,
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
	switch msg := msg.(type) {
	case InputSetValueMsg:
		p.textarea.SetValue(msg.Value)
		return p, nil
	case InputResetMsg:
		p.textarea.Reset()
		return p, nil
	case InputFocusMsg:
		p.focused = true
		p.textarea.Focus()
		return p, nil
	case InputBlurMsg:
		p.focused = false
		p.textarea.Blur()
		return p, nil
	case InputAddHistoryMsg:
		p.AddHistory(msg.Value)
		return p, nil
	case InputSetCompletionsMsg:
		p.SetCompletions(msg.Names)
		return p, nil
	case InputSetPlaceholderMsg:
		p.textarea.Placeholder = msg.Text
		return p, nil
	case ResizeMsg:
		if msg.Width > 0 {
			p.textarea.SetWidth(msg.Width)
		}
		if msg.Height > 0 {
			p.textarea.SetHeight(msg.Height)
		}
		return p, nil
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		// Enter emits SubmitMsg
		if msg.Type == tea.KeyEnter && !msg.Alt {
			val := strings.TrimSpace(p.textarea.Value())
			if val != "" {
				p.textarea.Reset()
				return p, func() tea.Msg {
					return SubmitMsg{Value: val}
				}
			}
			return p, nil
		}
	}
	if !p.focused {
		return p, nil
	}
	var cmd tea.Cmd
	p.textarea, cmd = p.textarea.Update(msg)
	return p, cmd
}

func (p *InputPanel) View(width int) string {
	if !p.visible {
		return ""
	}
	if width > 0 {
		p.textarea.SetWidth(width)
	}
	return p.textarea.View()
}

// SetVisible controls whether the input panel renders.
func (p *InputPanel) SetVisible(v bool) { p.visible = v }

// Visible returns whether the input panel is visible.
func (p *InputPanel) Visible() bool { return p.visible }

// SetCompletions configures the sorted command names for tab completion.
func (p *InputPanel) SetCompletions(names []string) {
	p.completions = names
}

// TabComplete attempts slash command completion. Returns (handled, cmd).
// If there's exactly one match, auto-executes it via SubmitMsg.
func (p *InputPanel) TabComplete() (bool, tea.Cmd) {
	val := p.textarea.Value()
	if !strings.HasPrefix(val, "/") {
		p.compPrefix = ""
		p.lastCompletion = ""
		return false, nil
	}

	// Determine the prefix to match against.
	if val == p.lastCompletion && p.compPrefix != "" {
		// User pressed Tab again on a completed value — cycle to next match.
	} else {
		// New prefix — start fresh.
		p.compPrefix = val
		p.compIdx = -1
		p.matches = filterPrefix(p.completions, val)
	}

	if len(p.matches) == 0 {
		return true, nil // consumed Tab but no matches
	}

	// Single match and first completion — auto-execute.
	if len(p.matches) == 1 && p.lastCompletion == "" {
		completed := p.matches[0]
		p.textarea.Reset()
		p.compPrefix = ""
		p.lastCompletion = ""
		return true, func() tea.Msg {
			return SubmitMsg{Value: completed}
		}
	}

	p.compIdx = (p.compIdx + 1) % len(p.matches)
	completed := p.matches[p.compIdx]
	p.textarea.SetValue(completed)
	p.lastCompletion = completed
	return true, nil
}

func filterPrefix(names []string, prefix string) []string {
	var out []string
	for _, n := range names {
		if strings.HasPrefix(n, prefix) {
			out = append(out, n)
		}
	}
	return out
}
