package repl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
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
	store        *session.Store // auto-save after each turn
	enforcer     policy.Enforcer
	token        policy.CapabilityToken
	log          *slog.Logger
	ctx          context.Context

	// UI state
	state        State
	textInput    textarea.Model
	vp           viewport.Model
	vpReady      bool
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
	healthReports  []tui.HealthReport // component health for status line
	spin           spinner.Model
	spinnerActive  bool
	activeToolIdx  int  // conversation index of active tool spinner (-1 = none)
	outputPanel    *tui.OutputPanel
	dashboard      *tui.DashboardPanel
	focus          *tui.FocusManager
	ready          bool
	quitting       bool
	initialPrompt  string // auto-submit on first render
}

// NewModel creates a new REPL model.
func NewModel(cfg Config) Model {
	ti := textarea.New()
	ti.Prompt = tui.UserStyle.Render(tui.LabelUser)
	ti.ShowLineNumbers = false
	ti.SetHeight(1)
	ti.CharLimit = 0 // unlimited
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

	m := Model{
		chatDriver:   cfg.Driver,
		tools:        cfg.Tools,
		sess:         cfg.Session,
		systemPrompt: cfg.SystemPrompt,
		maxTurns:     cfg.MaxTurns,
		autoApprove:  cfg.AutoApprove,
		mode:         mode,
		approvalCh:   make(chan bool, 1),
		store:        cfg.Store,
		enforcer:     cfg.Enforcer,
		token:        cfg.Token,
		log:          log,
		ctx:          context.Background(),
		state:        stateInput,
		textInput:    ti,
		historyIdx:    -1,
		handler:       agent.NilHandler{},
		healthReports:  cfg.HealthReports,
		spin:           spinner.New(spinner.WithSpinner(spinner.Dot)),
		activeToolIdx:  -1,
		outputPanel:    tui.NewOutputPanel(),
		dashboard:      tui.NewDashboardPanel(),
		initialPrompt:  cfg.InitialPrompt,
	}

	m.focus = tui.NewFocusManager(m.outputPanel, m.dashboard)
	m.dashboard.SetIdentity(cfg.Session.Workspace, cfg.Session.Driver, cfg.Session.Model, cfg.Mode)
	m.dashboard.SetHealth(cfg.HealthReports)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.SetWindowTitle("djinn"),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		tui.ReinitRenderer(msg.Width)
		// Reserve 3 lines: input + status + padding
		vpHeight := msg.Height - 3
		if vpHeight < 5 {
			vpHeight = 5
		}
		if !m.vpReady {
			m.vp = viewport.New(msg.Width, vpHeight)
			m.vpReady = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = vpHeight
		}
		// Auto-submit initial prompt if provided
		if m.initialPrompt != "" {
			m.textInput.SetValue(m.initialPrompt)
			m.initialPrompt = "" // only once
			return m.handleSubmit()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		if m.spinnerActive {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case tui.TextMsg:
		m.spinnerActive = false // first token arrived, hide spinner
		if m.outputMode == outputChunked {
			m.chunkedBuf.WriteString(string(msg))
		} else {
			m.streamBuf.WriteString(string(msg))
		}
		return m, nil

	case tui.ThinkingMsg:
		m.conversation = append(m.conversation,
			tui.DimStyle.Render(string(msg)))
		return m, nil

	case tui.ToolCallMsg:
		line := fmt.Sprintf("  %s %s %s",
			m.spin.View(),
			tui.ToolNameStyle.Render(msg.Call.Name),
			tui.ToolArgStyle.Render(truncate(string(msg.Call.Input), 80)))
		m.conversation = append(m.conversation, line)
		m.activeToolIdx = len(m.conversation) - 1

		// In agent mode (not auto), prompt for approval
		if m.mode == agent.ModeAgent && !m.autoApprove {
			m.state = stateToolApproval
			m.pendingTool = &msg.Call
		}
		return m, nil

	case tui.ToolResultMsg:
		var line string
		if msg.IsError {
			line = fmt.Sprintf("  %s %s",
				tui.ErrorStyle.Render(msg.Name),
				tui.DimStyle.Render(truncate(msg.Output, 100)))
		} else {
			lines := strings.Count(msg.Output, "\n")
			if lines > 3 {
				line = fmt.Sprintf("  %s (%d lines)",
					tui.ToolSuccessStyle.Render("✓ "+msg.Name), lines)
			} else {
				line = fmt.Sprintf("  %s %s",
					tui.ToolSuccessStyle.Render("✓ "+msg.Name),
					tui.DimStyle.Render(truncate(msg.Output, 100)))
			}
		}
		// Replace spinner line with result if we have an active tool
		if m.activeToolIdx >= 0 && m.activeToolIdx < len(m.conversation) {
			m.conversation[m.activeToolIdx] = line
			m.activeToolIdx = -1
		} else {
			m.conversation = append(m.conversation, line)
		}
		return m, nil

	case tui.DoneMsg:
		m.lastUsage = msg.Usage
		if msg.Usage != nil {
			m.totalIn += msg.Usage.InputTokens
			m.totalOut += msg.Usage.OutputTokens
		}
		return m, nil

	case tui.ErrorMsg:
		m.lastError = msg.Error()
		m.conversation = append(m.conversation,
			tui.ErrorStyle.Render("error: "+msg.Error()))
		return m, nil

	case tui.AgentDoneMsg:
		// Auto-save session after each turn
		if m.store != nil {
			if err := m.store.Save(m.sess); err != nil {
				m.log.Warn("auto-save failed", "error", err)
			}
		}
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
		// Render the completed response as markdown
		if len(m.conversation) > 0 {
			last := len(m.conversation) - 1
			raw := m.conversation[last]
			// Strip the assistant label prefix, render, re-add
			prefix := tui.AssistStyle.Render(tui.LabelAssist) + ": "
			if after, found := strings.CutPrefix(raw, prefix); found {
				rendered := tui.RenderMarkdown(after)
				m.conversation[last] = prefix + rendered
			}
		}
		if msg.Err != nil {
			m.conversation = append(m.conversation,
				tui.ErrorStyle.Render("error: "+msg.Err.Error()))
		}
		if m.lastUsage != nil {
			m.conversation = append(m.conversation,
				tui.StatusStyle.Render(fmt.Sprintf("[tokens: %d in, %d out]",
					m.lastUsage.InputTokens, m.lastUsage.OutputTokens)))
		}
		m.conversation = append(m.conversation, "") // blank line
		m.state = stateInput
		m.textInput.Focus()
		return m, nil

	case tui.TickMsg:
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

	// Build conversation content for viewport
	var content strings.Builder

	// Welcome header (first time)
	if len(m.conversation) == 0 && m.state == stateInput {
		content.WriteString(renderMOTD(m.sess, m.tools, m.healthReports, m.width))
		content.WriteString("\n")
	}

	// Conversation history (word-wrapped)
	for _, line := range m.conversation {
		if m.width > 0 {
			content.WriteString(tui.WrapText(line, m.width-2))
		} else {
			content.WriteString(line)
		}
		content.WriteString("\n")
	}

	// Spinner or streaming text
	if m.state == stateStreaming {
		if m.spinnerActive {
			content.WriteString("  " + m.spin.View() + " thinking...\n")
		} else if m.streamBuf.Len() > 0 {
			content.WriteString(m.streamBuf.String())
		}
	}

	// Tool approval prompt
	if m.state == stateToolApproval && m.pendingTool != nil {
		content.WriteString(tui.ToolNameStyle.Render(
			fmt.Sprintf("  approve %s? [y/n] ", m.pendingTool.Name)))
	}

	// Sync output panel with conversation content
	m.outputPanel.InitViewport(m.width, m.height-3)

	// Render output through panel
	depths := tui.FocusDepths(m.focus.Count(), m.focus.ActiveIndex())
	outputView := content.String()
	if m.vpReady {
		m.vp.SetContent(outputView)
		m.vp.GotoBottom()
		outputView = m.vp.View()
	}
	sb.WriteString(tui.RenderWithDepth(outputView, depths[0]))

	// Separator: output ↔ input (with focus indicator)
	sb.WriteString("\n")
	sb.WriteString(tui.RenderFocusIndicator(m.focus.ActiveIndex() == 0))
	sb.WriteString(tui.Separator(m.width-1, 0, m.focus.ActiveIndex() == 0))
	sb.WriteString("\n")

	// Input
	if m.state == stateInput {
		sb.WriteString(m.textInput.View())
	}

	// Separator: input ↔ dashboard (with focus indicator)
	sb.WriteString("\n")
	sb.WriteString(tui.RenderFocusIndicator(m.focus.ActiveIndex() == 1))
	sb.WriteString(tui.Separator(m.width-1, 0, m.focus.ActiveIndex() == 1))
	sb.WriteString("\n")

	// Dashboard
	if m.ready && !m.quitting {
		m.dashboard.SetMetrics(m.totalIn, m.totalOut, m.sess.History.Len())
		sb.WriteString(tui.RenderWithDepth(m.dashboard.View(m.width), depths[len(depths)-1]))
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
			if msg.Alt {
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
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

	// Tab cycles focus between panels
	if msg.Type == tea.KeyTab && m.state == stateInput {
		m.focus.Cycle()
		return m, nil
	}

	// Forward PgUp/PgDn to viewport for scrolling
	if m.vpReady && (msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown) {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}

	if m.state == stateInput {
		// Alt+M cycles mode: ask → plan → agent → auto → ask
		if msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'm' || msg.Runes[0] == 'M') {
			m.mode = m.mode.Next()
			m.sess.Mode = m.mode.String()
			m.conversation = append(m.conversation,
				tui.DimStyle.Render(fmt.Sprintf("  mode: %s", m.mode)))
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
				// cursor moves to end naturally with SetValue
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
					// cursor moves to end naturally with SetValue
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
	m.conversation = append(m.conversation, tui.UserStyle.Render(tui.LabelUser)+input)

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
	m.spinnerActive = true
	m.conversation = append(m.conversation, tui.AssistStyle.Render(tui.LabelAssist)+": ")
	m.textInput.Blur()

	return m, tea.Batch(
		m.runAgent(input),
		m.spin.Tick,
		tickCmd(),
	)
}

func (m *Model) handleApproval(key string) (tea.Model, tea.Cmd) {
	approved := key == "y" || key == "Y"
	m.pendingTool = nil
	m.state = stateStreaming

	if approved {
		m.conversation = append(m.conversation, tui.DimStyle.Render("  approved"))
	} else {
		m.conversation = append(m.conversation, tui.ErrorStyle.Render("  denied"))
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
			Enforcer:     m.enforcer,
			Token:        m.token,
			Handler:      globalHandler,
			Log:          agentLog,
		}, prompt)
		return tui.AgentDoneMsg{Result: result, Err: err}
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

// renderMOTD builds the welcome banner with logo and workspace info.
func renderMOTD(sess *session.Session, tools *builtin.Registry, health []tui.HealthReport, width int) string {
	logo := tui.LogoStyle.Render(tui.DjinnLogo)

	wsName := sess.Workspace
	if wsName == "" {
		wsName = "(ephemeral)"
	}

	var info strings.Builder
	info.WriteString(tui.AssistStyle.Render("Djinn v0.1.0") + "\n\n")
	info.WriteString(fmt.Sprintf("  ws:    %s\n", wsName))
	info.WriteString(fmt.Sprintf("  model: %s\n", sess.Model))
	mode := sess.Mode
	if mode == "" {
		mode = "agent"
	}
	info.WriteString(fmt.Sprintf("  mode:  %s\n", mode))
	info.WriteString(fmt.Sprintf("  tools: %d\n", len(tools.Names())))
	info.WriteString("\n")
	info.WriteString(tui.DimStyle.Render("  /help for commands"))
	info.WriteString("\n")
	info.WriteString(tui.DimStyle.Render("  Tab to cycle panels"))

	// Join logo (left) + info (right)
	return lipgloss.JoinHorizontal(lipgloss.Top, logo+"   ", info.String())
}

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
		return tui.TickMsg(t)
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
