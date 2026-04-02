// input.go — InputPanel wraps textarea with focus indicator and history.
package widgets

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/tui/core"
	"github.com/dpopsuev/djinn/tui/layout"

	tui "github.com/dpopsuev/djinn/tui"
)

// InputPanel is the user input area.
type InputPanel struct {
	core.BasePanel
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

	// Predictive input — grey suggestion from history.
	prediction string

	// Sandbox state — changes prompt from > to [>].
	sandboxed bool
}

const panelIDInput = "input"

var _ core.Panel = (*InputPanel)(nil)

// InputOption configures an InputPanel.
type InputOption func(*InputPanel)

// WithPlaceholder sets the input placeholder text.
func WithPlaceholder(text string) InputOption {
	return func(p *InputPanel) { p.textarea.Placeholder = text }
}

// WithCompletions sets the tab-completion command names.
func WithCompletions(names []string) InputOption {
	return func(p *InputPanel) { p.completions = names }
}

// WithOnSubmit sets the submit callback.
func WithOnSubmit(fn func(string)) InputOption {
	return func(p *InputPanel) { p.onSubmit = fn }
}

// NewInputPanel creates the input panel.
func NewInputPanel(opts ...InputOption) *InputPanel {
	ta := textarea.New()
	ta.Prompt = "" // No per-line prompt — chevron prepended in View() on first line only.
	ta.Placeholder = `Try "explain this codebase"`
	ta.ShowLineNumbers = false
	ta.SetHeight(1) // Single line by default — expands upward on multi-line input.
	ta.CharLimit = 0

	// Clear all backgrounds — our design language is foreground-only.
	// The textarea component ships with default backgrounds that break
	// terminal transparency.
	noBackground := lipgloss.NewStyle()
	ta.FocusedStyle.Base = noBackground
	ta.BlurredStyle.Base = noBackground
	ta.FocusedStyle.CursorLine = noBackground
	ta.BlurredStyle.CursorLine = noBackground
	ta.FocusedStyle.Placeholder = tui.DimStyle
	ta.BlurredStyle.Placeholder = tui.DimStyle

	// User input text in green (BUG-30). No background.
	ta.FocusedStyle.Text = tui.UserStyle
	ta.BlurredStyle.Text = tui.DimStyle
	ta.Focus()

	p := &InputPanel{
		BasePanel: core.NewBasePanel(panelIDInput, 1),
		textarea:  ta,
		histIdx:   -1,
		visible:   true,
		compIdx:   -1,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
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
	p.BasePanel.SetFocus(f)
	if f {
		p.textarea.Focus()
	} else {
		p.textarea.Blur()
	}
}

func (p *InputPanel) Update(msg tea.Msg) (core.Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.InputSetValueMsg:
		p.textarea.SetValue(msg.Value)
		return p, nil
	case tui.InputResetMsg:
		p.textarea.Reset()
		return p, nil
	case tui.InputFocusMsg:
		p.SetFocus(true)
		p.textarea.Focus()
		return p, nil
	case tui.InputBlurMsg:
		p.SetFocus(false)
		p.textarea.Blur()
		return p, nil
	case tui.InputAddHistoryMsg:
		p.AddHistory(msg.Value)
		return p, nil
	case tui.InputSetCompletionsMsg:
		p.SetCompletions(msg.Names)
		return p, nil
	case tui.InputSetPlaceholderMsg:
		p.textarea.Placeholder = msg.Text
		return p, nil
	case tui.SandboxStateMsg:
		p.sandboxed = msg.Sandboxed
		return p, nil
	case layout.ResizeMsg:
		if msg.Width > 0 {
			p.textarea.SetWidth(msg.Width)
		}
		if msg.Height > 0 {
			p.textarea.SetHeight(msg.Height)
		}
		return p, nil
	case tea.KeyMsg:
		if !p.Focused() {
			return p, nil
		}
		// Enter emits SubmitMsg
		if msg.Type == tea.KeyEnter && !msg.Alt {
			val := strings.TrimSpace(p.textarea.Value())
			if val != "" {
				p.textarea.Reset()
				return p, func() tea.Msg {
					return tui.SubmitMsg{Value: val}
				}
			}
			return p, nil
		}
	}
	if !p.Focused() {
		return p, nil
	}
	var cmd tea.Cmd
	p.textarea, cmd = p.textarea.Update(msg)
	// Update prediction after each keystroke.
	p.updatePrediction()
	return p, cmd
}

// updatePrediction searches history for a prefix match.
func (p *InputPanel) updatePrediction() {
	val := p.textarea.Value()
	p.prediction = ""
	if val == "" || strings.HasPrefix(val, "/") {
		return
	}
	for i := len(p.history) - 1; i >= 0; i-- {
		if strings.HasPrefix(p.history[i], val) && p.history[i] != val {
			p.prediction = p.history[i]
			return
		}
	}
}

// AcceptPrediction sets the input value to the current prediction.
func (p *InputPanel) AcceptPrediction() bool {
	if p.prediction == "" {
		return false
	}
	p.textarea.SetValue(p.prediction)
	p.prediction = ""
	return true
}

// Prediction returns the current prediction text (for testing).
func (p *InputPanel) Prediction() string {
	return p.prediction
}

func (p *InputPanel) View(width int) string {
	if !p.visible {
		return ""
	}
	prompt := tui.LabelUser
	if p.sandboxed {
		prompt = tui.ActiveGlyphs.Sandboxed + " "
	}
	promptPrefix := tui.UserStyle.Render(prompt)
	if width > 0 {
		// Account for prompt width in textarea.
		p.textarea.SetWidth(width - len([]rune(prompt)))
	}
	view := promptPrefix + p.textarea.View()

	// Show prediction as dim suffix.
	if p.prediction != "" {
		val := p.textarea.Value()
		if strings.HasPrefix(p.prediction, val) {
			suffix := p.prediction[len(val):]
			if suffix != "" {
				view += tui.DimStyle.Render(suffix)
			}
		}
	}

	// Show slash command hints when typing /.
	val := p.textarea.Value()
	if strings.HasPrefix(val, "/") && len(p.completions) > 0 {
		matches := filterPrefix(p.completions, val)
		if len(matches) > 0 && !(len(matches) == 1 && matches[0] == val) {
			view += "\n" + tui.DimStyle.Render(strings.Join(matches, "  "))
		}
	}

	return view
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
			return tui.SubmitMsg{Value: completed}
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
