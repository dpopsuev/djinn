package repl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
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

// Model is the Bubbletea model for the Djinn REPL.
type Model struct {
	// Dependencies
	apiDriver    *claudedriver.APIDriver
	tools        *builtin.Registry
	sess         *session.Session
	systemPrompt string
	maxTurns     int
	autoApprove  bool
	ctx          context.Context

	// UI state
	state        State
	textInput    textinput.Model
	streamBuf    strings.Builder
	conversation []string // rendered conversation lines
	pendingTool  *driver.ToolCall
	lastUsage    *driver.Usage
	lastError    string
	handler      agent.EventHandler
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

	return Model{
		apiDriver:    cfg.Driver,
		tools:        cfg.Tools,
		sess:         cfg.Session,
		systemPrompt: cfg.SystemPrompt,
		maxTurns:     cfg.MaxTurns,
		autoApprove:  cfg.AutoApprove,
		ctx:          context.Background(),
		state:        stateInput,
		textInput:    ti,
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
		m.streamBuf.WriteString(string(msg))
		return m, nil // wait for tick to render

	case ThinkingMsg:
		m.conversation = append(m.conversation,
			dimStyle.Render(string(msg)))
		return m, nil

	case ToolCallMsg:
		line := fmt.Sprintf("  %s %s",
			toolNameStyle.Render(msg.Call.Name),
			toolArgStyle.Render(truncate(string(msg.Call.Input), 80)))
		m.conversation = append(m.conversation, line)

		if !m.autoApprove {
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
		return m, nil

	case ErrorMsg:
		m.lastError = msg.Error()
		m.conversation = append(m.conversation,
			errorStyle.Render("error: "+msg.Error()))
		return m, nil

	case AgentDoneMsg:
		// Flush remaining stream buffer
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
		sb.WriteString(assistStyle.Render("Djinn REPL"))
		sb.WriteString(dimStyle.Render(fmt.Sprintf(" — model: %s — tools: %d — /help for commands",
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
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())
	m.textInput.Reset()

	if input == "" {
		return m, nil
	}

	// Add user input to conversation
	m.conversation = append(m.conversation, userStyle.Render(labelUser)+input)

	// Check for slash command
	if cmd, ok := ParseCommand(input); ok {
		result := ExecuteCommand(cmd, m.sess)
		if result.Output != "" {
			m.conversation = append(m.conversation, result.Output)
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

	return m, nil
}

func (m *Model) runAgent(prompt string) tea.Cmd {
	return func() tea.Msg {
		// The handler will be set after program starts — use a placeholder
		// that sends messages back to the program. This is handled by
		// the Run() function which creates the handler after tea.NewProgram.
		result, err := agent.Run(m.ctx, agent.Config{
			Driver:       m.apiDriver,
			Tools:        m.tools,
			Session:      m.sess,
			SystemPrompt: m.systemPrompt,
			MaxTurns:     m.maxTurns,
			Approve:      agent.AutoApprove,
			Handler:      globalHandler,
		}, prompt)
		return AgentDoneMsg{Result: result, Err: err}
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
