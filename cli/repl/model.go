package repl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// Frame rate for smooth streaming (60fps).
const tickInterval = 16 * time.Millisecond

// State represents the REPL state machine.
type State int

const (
	stateInput State = iota
	stateStreaming
	stateToolApproval
)

// OutputMode controls how agent responses are rendered.
type OutputMode int

const (
	outputStreaming OutputMode = iota // token-by-token (default)
	outputChunked                    // all-at-once after completion
)

// ScrollMode controls auto-scroll behavior.
type ScrollMode int

const (
	scrollFollow ScrollMode = iota // always show latest (default)
	scrollStatic                   // user scrolls manually
)

// Model is the Bubbletea model for the Djinn REPL.
type Model struct {
	// Dependencies
	chatDriver   driver.ChatDriver
	tools        *builtin.Registry
	sess         *session.Session
	systemPrompt string
	maxTurns     int
	autoApprove  bool
	mode         agent.Mode
	approvalCh   chan bool // bridges approval from UI to agent goroutine
	log          *slog.Logger
	ctx          context.Context

	// UI state
	state        State
	textInput    textinput.Model
	streamBuf    strings.Builder
	conversation []string // rendered conversation lines
	inputHistory []string // previous prompts
	historyIdx   int      // -1 = not browsing history
	pendingTool  *driver.ToolCall
	lastUsage    *driver.Usage
	totalIn      int      // cumulative input tokens
	totalOut     int      // cumulative output tokens
	lastError    string
	handler      agent.EventHandler
	outputMode   OutputMode
	scrollMode   ScrollMode
	chunkedBuf   strings.Builder // accumulates full response for chunked mode
	width        int
	height       int
	ready        bool
	quitting     bool
}

// NewModel creates a new REPL model.
func NewModel(cfg Config) Model {
	ti := textinput.New()
	ti.Prompt = userStyle.Render(labelUser)
	ti.Focus()

	mode, err := agent.ParseMode(cfg.Mode)
	if err != nil {
		mode = agent.ModeAgent
	}

	log := cfg.Log
	if log == nil {
		log = djinnlog.Nop()
	}
	log = djinnlog.For(log, "repl")

	return Model{
		chatDriver:   cfg.Driver,
		tools:        cfg.Tools,
		sess:         cfg.Session,
		systemPrompt: cfg.SystemPrompt,
		maxTurns:     cfg.MaxTurns,
		autoApprove:  cfg.AutoApprove,
		mode:         mode,
		approvalCh:   make(chan bool, 1),
		log:          log,
		ctx:          context.Background(),
		state:        stateInput,
		textInput:    ti,
		historyIdx:   -1,
		handler:      agent.NilHandler{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.SetWindowTitle("djinn"),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case TextMsg:
		if m.outputMode == outputChunked {
			m.chunkedBuf.WriteString(string(msg))
		} else {
			m.streamBuf.WriteString(string(msg))
		}
		return m, nil // wait for tick to render (streaming) or done (chunked)

	case ThinkingMsg:
		m.conversation = append(m.conversation,
			dimStyle.Render(string(msg)))
		return m, nil

	case ToolCallMsg:
		line := fmt.Sprintf("  %s %s",
			toolNameStyle.Render(msg.Call.Name),
			toolArgStyle.Render(truncate(string(msg.Call.Input), 80)))
		m.conversation = append(m.conversation, line)

		// In agent mode (not auto), prompt for approval
		if m.mode == agent.ModeAgent && !m.autoApprove {
			m.state = stateToolApproval
			m.pendingTool = &msg.Call
		}
		return m, nil

	case ToolResultMsg:
		var line string
		if msg.IsError {
			line = fmt.Sprintf("  %s %s",
				errorStyle.Render(msg.Name),
				dimStyle.Render(truncate(msg.Output, 100)))
		} else {
			lines := strings.Count(msg.Output, "\n")
			if lines > 3 {
				line = fmt.Sprintf("  %s (%d lines)",
					toolSuccessStyle.Render(msg.Name), lines)
			} else {
				line = fmt.Sprintf("  %s %s",
					toolSuccessStyle.Render(msg.Name),
					dimStyle.Render(truncate(msg.Output, 100)))
			}
		}
		m.conversation = append(m.conversation, line)
		return m, nil

	case DoneMsg:
		m.lastUsage = msg.Usage
		if msg.Usage != nil {
			m.totalIn += msg.Usage.InputTokens
			m.totalOut += msg.Usage.OutputTokens
		}
		return m, nil

	case ErrorMsg:
		m.lastError = msg.Error()
		m.conversation = append(m.conversation,
			errorStyle.Render("error: "+msg.Error()))
		return m, nil

	case AgentDoneMsg:
		// Flush remaining buffers
		if m.outputMode == outputChunked && m.chunkedBuf.Len() > 0 {
			// Chunked mode: render full response now
			if len(m.conversation) > 0 {
				last := len(m.conversation) - 1
				m.conversation[last] += m.chunkedBuf.String()
			}
			m.chunkedBuf.Reset()
		}
		m.flushStreamBuffer()
		if msg.Err != nil {
			m.conversation = append(m.conversation,
				errorStyle.Render("error: "+msg.Err.Error()))
		}
		if m.lastUsage != nil {
			m.conversation = append(m.conversation,
				statusStyle.Render(fmt.Sprintf("[tokens: %d in, %d out]",
					m.lastUsage.InputTokens, m.lastUsage.OutputTokens)))
		}
		m.conversation = append(m.conversation, "") // blank line
		m.state = stateInput
		m.textInput.Focus()
		return m, nil

	case TickMsg:
		if m.state == stateStreaming {
			m.flushStreamBuffer()
			return m, tickCmd()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "initializing..."
	}

	var sb strings.Builder

	// Welcome header (first time)
	if len(m.conversation) == 0 && m.state == stateInput {
		sb.WriteString(logoStyle.Render(djinnLogo))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  model: %s — tools: %d — /help for commands",
			m.sess.Model, len(m.tools.Names()))))
		sb.WriteString("\n\n")
	}

	// Conversation history
	for _, line := range m.conversation {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Current streaming text
	if m.state == stateStreaming && m.streamBuf.Len() > 0 {
		sb.WriteString(m.streamBuf.String())
	}

	// Tool approval prompt
	if m.state == stateToolApproval && m.pendingTool != nil {
		sb.WriteString(toolNameStyle.Render(
			fmt.Sprintf("  approve %s? [y/n] ", m.pendingTool.Name)))
	}

	// Input
	if m.state == stateInput {
		sb.WriteString(m.textInput.View())
	}

	// Status bar
	if m.ready && !m.quitting {
		sb.WriteString("\n")
		status := fmt.Sprintf("  %s │ %s │ tokens: %d in, %d out │ turns: %d",
			m.sess.Model, m.mode, m.totalIn, m.totalOut, m.sess.History.Len())
		if m.sess.Name != "" {
			status += " │ " + m.sess.Name
		}
		sb.WriteString(statusStyle.Render(status))
	}

	if m.quitting {
		sb.WriteString("\ngoodbye.\n")
	}

	return sb.String()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEnter:
		if m.state == stateToolApproval {
			return m.handleApproval(msg.String())
		}
		if m.state == stateInput {
			return m.handleSubmit()
		}
	}

	if m.state == stateToolApproval {
		// Single key approval
		switch msg.String() {
		case "y", "Y":
			return m.handleApproval("y")
		case "n", "N":
			return m.handleApproval("n")
		}
		return m, nil
	}

	if m.state == stateInput {
		// Alt+M cycles mode: ask → plan → agent → auto → ask
		if msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'm' || msg.Runes[0] == 'M') {
			m.mode = m.mode.Next()
			m.sess.Mode = m.mode.String()
			m.conversation = append(m.conversation,
				dimStyle.Render(fmt.Sprintf("  mode: %s", m.mode)))
			return m, nil
		}

		// Input history navigation
		switch msg.Type {
		case tea.KeyUp:
			if len(m.inputHistory) > 0 {
				if m.historyIdx == -1 {
					m.historyIdx = len(m.inputHistory) - 1
				} else if m.historyIdx > 0 {
					m.historyIdx--
				}
				m.textInput.SetValue(m.inputHistory[m.historyIdx])
				m.textInput.CursorEnd()
			}
			return m, nil
		case tea.KeyDown:
			if m.historyIdx >= 0 {
				m.historyIdx++
				if m.historyIdx >= len(m.inputHistory) {
					m.historyIdx = -1
					m.textInput.SetValue("")
				} else {
					m.textInput.SetValue(m.inputHistory[m.historyIdx])
					m.textInput.CursorEnd()
				}
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())
	m.textInput.Reset()
	m.historyIdx = -1 // reset history browsing

	if input == "" {
		return m, nil
	}

	// Record in input history
	m.inputHistory = append(m.inputHistory, input)

	// Add user input to conversation
	m.conversation = append(m.conversation, userStyle.Render(labelUser)+input)

	// Check for slash command
	if cmd, ok := ParseCommand(input); ok {
		result := ExecuteCommand(cmd, m.sess)
		if result.Output != "" {
			m.conversation = append(m.conversation, result.Output)
		}
		if result.ModeChange != "" {
			if newMode, err := agent.ParseMode(result.ModeChange); err == nil {
				m.mode = newMode
			}
		}
		if result.Exit {
			m.quitting = true
			return m, tea.Quit
		}
		m.conversation = append(m.conversation, "") // blank line
		return m, nil
	}

	// Start agent loop
	m.state = stateStreaming
	m.streamBuf.Reset()
	m.lastUsage = nil
	m.lastError = ""
	m.conversation = append(m.conversation, assistStyle.Render(labelAssist)+": ")
	m.textInput.Blur()

	return m, tea.Batch(
		m.runAgent(input),
		tickCmd(),
	)
}

func (m *Model) handleApproval(key string) (tea.Model, tea.Cmd) {
	approved := key == "y" || key == "Y"
	m.pendingTool = nil
	m.state = stateStreaming

	if approved {
		m.conversation = append(m.conversation, dimStyle.Render("  approved"))
	} else {
		m.conversation = append(m.conversation, errorStyle.Render("  denied"))
	}

	// Send decision to agent goroutine via channel
	ch := m.approvalCh
	return m, func() tea.Msg {
		ch <- approved
		return nil
	}
}

func (m *Model) runAgent(prompt string) tea.Cmd {
	mode := m.mode
	ch := m.approvalCh
	agentLog := djinnlog.For(m.log, "agent")
	return func() tea.Msg {
		result, err := agent.Run(m.ctx, agent.Config{
			Driver:       m.chatDriver,
			Tools:        m.tools,
			Session:      m.sess,
			SystemPrompt: m.systemPrompt,
			MaxTurns:     m.maxTurns,
			ToolsEnabled: mode.ToolsEnabled(),
			Mode:         mode,
			Approve:      approvalForMode(mode, ch),
			Handler:      globalHandler,
			Log:          agentLog,
		}, prompt)
		return AgentDoneMsg{Result: result, Err: err}
	}
}

// approvalForMode returns the approval function for the given mode.
// In agent mode, blocks on the channel waiting for the UI decision.
func approvalForMode(mode agent.Mode, ch chan bool) agent.ApprovalFunc {
	switch mode {
	case agent.ModeAuto:
		return agent.AutoApprove
	case agent.ModeAgent:
		return func(_ driver.ToolCall) bool {
			return <-ch
		}
	default:
		return agent.DenyAll
	}
}

func (m *Model) flushStreamBuffer() {
	if m.streamBuf.Len() == 0 {
		return
	}
	// Move buffered text into conversation as part of assistant message
	text := m.streamBuf.String()
	m.streamBuf.Reset()

	// Append to the last conversation line (the assistant label)
	if len(m.conversation) > 0 {
		last := len(m.conversation) - 1
		m.conversation[last] += text
	}
}

// Test accessors — exported for acceptance tests.

// SetTextInput sets the text input value (for testing).
func (m *Model) SetTextInput(v string) { m.textInput.SetValue(v) }

// SetState sets the model state (for testing).
func (m *Model) SetState(s State) { m.state = s }

// AppendConversation adds a line to conversation (for testing).
func (m *Model) AppendConversation(line string) { m.conversation = append(m.conversation, line) }

// StreamBufString returns the stream buffer contents (for testing).
func (m Model) StreamBufString() string { return m.streamBuf.String() }

// CurrentState returns the current state (for testing).
func (m Model) CurrentState() State { return m.state }

// TextInputValue returns the text input value (for testing).
func (m Model) TextInputValue() string { return m.textInput.Value() }

// AddInputHistory adds an entry to input history (for testing).
func (m *Model) AddInputHistory(s string) { m.inputHistory = append(m.inputHistory, s) }

// Export state constants for acceptance tests.
const (
	StateInput        = stateInput
	StateStreaming     = stateStreaming
	StateToolApproval = stateToolApproval
)

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
