package repl

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
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
	state       State
	inputPanel  *tui.InputPanel
	streamBuf   *strings.Builder
	pendingTool *driver.ToolCall
	lastUsage    *driver.Usage
	totalIn      int      // cumulative input tokens
	totalOut     int      // cumulative output tokens
	lastError    string
	handler      agent.EventHandler
	outputMode   OutputMode
	chunkedBuf   *strings.Builder // accumulates full response for chunked mode
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
	version        string // app version for MOTD
	rawStreamLine  *strings.Builder // raw unrendered text for incremental markdown (pointer: Builder can't be copied)

	// Debug
	debugTap     *tui.DebugTap

	// Staff — role pipeline
	currentRole  string
	roleMemory   *staff.RoleMemory
	roles        map[string]staff.Role
}

// NewModel creates a new REPL model.
func NewModel(cfg Config) Model {
	inputPanel := tui.NewInputPanel()
	inputPanel.SetCompletions(CommandNames())

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
		inputPanel:   inputPanel,
		handler:       agent.NilHandler{},
		healthReports:  cfg.HealthReports,
		spin:           spinner.New(spinner.WithSpinner(spinner.Dot)),
		activeToolIdx:  -1,
		outputPanel:    tui.NewOutputPanel(),
		dashboard:      tui.NewDashboardPanel(),
		initialPrompt:  cfg.InitialPrompt,
		version:        cfg.Version,
		debugTap:       cfg.DebugTap,
		streamBuf:      &strings.Builder{},
		chunkedBuf:     &strings.Builder{},
		rawStreamLine:  &strings.Builder{},
	}

	m.focus = tui.NewFocusManager(m.outputPanel, m.inputPanel, m.dashboard)
	m.dashboard.SetIdentity(cfg.Session.Workspace, cfg.Session.Driver, cfg.Session.Model, cfg.Mode)
	m.dashboard.SetHealth(cfg.HealthReports)

	// Staff: initialize roles and memory.
	// The default role's mode overrides cfg.Mode — GenSec should be "plan"
	// (no tools) regardless of what the CLI flag says.
	staffCfg := staff.DefaultConfig()
	m.roles = staffCfg.RoleMap()
	m.roleMemory = staff.NewRoleMemory()
	m.currentRole = "gensec"
	if defaultRole, ok := m.roles["gensec"]; ok {
		if newMode, err := agent.ParseMode(defaultRole.Mode); err == nil {
			m.mode = newMode
		}
	}
	m.dashboard.SetIdentity(cfg.Session.Workspace, cfg.Session.Driver, cfg.Session.Model, m.mode.String())
	m.dashboard.SetUIState("GENSEC")

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
		m.outputPanel.InitViewport(msg.Width, msg.Height-m.fixedHeight())
		// Render MOTD once when viewport is first initialized
		if m.outputPanel.LineCount() == 0 {
			m.outputPanel.Append(renderMOTD(m.sess, m.tools, m.version, m.currentRole, msg.Width))
		}
		// Auto-submit initial prompt if provided
		if m.initialPrompt != "" {
			m.inputPanel.SetValue(m.initialPrompt)
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
		m.outputPanel.Append(tui.DimStyle.Render(string(msg)))
		return m, nil

	case tui.ToolCallMsg:
		line := fmt.Sprintf("  %s %s %s",
			m.spin.View(),
			tui.ToolNameStyle.Render(msg.Call.Name),
			tui.ToolArgStyle.Render(truncate(string(msg.Call.Input), 80)))
		m.outputPanel.Append(line)
		m.activeToolIdx = m.outputPanel.LineCount() - 1

		// In agent mode (not auto), prompt for approval
		if m.mode == agent.ModeAgent && !m.autoApprove {
			m.state = stateToolApproval
			m.dashboard.SetUIState("APPROVAL")
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
		if m.activeToolIdx >= 0 && m.activeToolIdx < m.outputPanel.LineCount() {
			m.outputPanel.SetLine(m.activeToolIdx, line)
			m.activeToolIdx = -1
		} else {
			m.outputPanel.Append(line)
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
		m.outputPanel.Append(tui.ErrorStyle.Render("error: " + msg.Error()))
		return m, nil

	case tui.AgentDoneMsg:
		m.rawStreamLine.Reset()
		// Auto-save session after each turn
		if m.store != nil {
			if err := m.store.Save(m.sess); err != nil {
				m.log.Warn("auto-save failed", "error", err)
			}
		}
		// Flush remaining buffers
		if m.outputMode == outputChunked && m.chunkedBuf.Len() > 0 {
			// Chunked mode: render full response now
			if m.outputPanel.LineCount() > 0 {
				last := m.outputPanel.LineCount() - 1
				m.outputPanel.SetLine(last, m.outputPanel.Lines()[last]+m.chunkedBuf.String())
			}
			m.chunkedBuf.Reset()
		}
		m.flushStreamBuffer()
		// Render the completed response as markdown
		if m.outputPanel.LineCount() > 0 {
			last := m.outputPanel.LineCount() - 1
			raw := m.outputPanel.Lines()[last]
			// Strip the assistant label prefix, render, re-add
			prefix := tui.AssistStyle.Render(tui.LabelAssist) + ": "
			if after, found := strings.CutPrefix(raw, prefix); found {
				rendered := tui.RenderMarkdown(after)
				m.outputPanel.SetLine(last, prefix+rendered)
			}
		}
		if msg.Err != nil {
			m.outputPanel.Append(tui.ErrorStyle.Render("error: " + msg.Err.Error()))
		}
		if m.lastUsage != nil {
			m.outputPanel.Append(tui.StatusStyle.Render(fmt.Sprintf("[tokens: %d in, %d out]",
				m.lastUsage.InputTokens, m.lastUsage.OutputTokens)))
		}
		m.outputPanel.Append("") // blank line
		m.state = stateInput
		m.inputPanel.FocusInput()

		// Auto-transition: determine next role
		if m.currentRole == "executor" {
			next := staff.NextRole(staff.SignalGatePassed)
			m.switchRole(next)
		} else if m.currentRole != "gensec" {
			m.switchRole("gensec")
		} else {
			m.dashboard.SetUIState("GENSEC")
		}
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
	if m.quitting {
		return "goodbye.\n"
	}

	depths := tui.FocusDepths(m.focus.Count(), m.focus.ActiveIndex())

	// Set ephemeral overlay (spinner, streaming text, approval prompt)
	m.outputPanel.SetOverlay(m.overlayContent())
	m.outputPanel.InitViewport(m.width, m.height-m.fixedHeight())
	m.dashboard.SetMetrics(m.totalIn, m.totalOut, m.sess.History.Len())

	var sb strings.Builder
	// Output panel: NEVER depth-dimmed. Dimming viewport-padded whitespace
	// causes ANSI escape codes to show as visible text on terminals without
	// 24-bit color support (DJN-BUG-11).
	sb.WriteString(m.outputPanel.View(m.width))
	sb.WriteString(m.separator(0))
	// Input panel: bordered when focused, hidden when streaming.
	if m.state == stateInput {
		sb.WriteString(tui.RenderWithDepth(m.inputPanel.View(m.width), depths[1]))
	}
	sb.WriteString(m.separator(1))
	// Dashboard: dimmed when unfocused.
	sb.WriteString(tui.RenderWithDepth(m.dashboard.View(m.width), depths[2]))

	result := sb.String()

	// DebugTap: capture every rendered frame
	if m.debugTap != nil {
		stateStr := "input"
		switch m.state {
		case stateStreaming:
			stateStr = "streaming"
		case stateToolApproval:
			stateStr = "approval"
		}
		m.debugTap.Capture(result, stateStr, m.currentRole, m.width, m.height)
	}

	return result
}

// overlayContent returns ephemeral content for the output panel based on state.
func (m Model) overlayContent() string {
	switch {
	case m.state == stateStreaming && m.spinnerActive:
		return "  " + m.spin.View() + " thinking..."
	case m.state == stateStreaming && m.streamBuf.Len() > 0:
		return m.streamBuf.String()
	case m.state == stateToolApproval && m.pendingTool != nil:
		return tui.ToolNameStyle.Render(fmt.Sprintf("  approve %s? [y/n] ", m.pendingTool.Name))
	default:
		return ""
	}
}

// separator renders a thin line between panels.
func (m Model) separator(_ int) string {
	return "\n"
}

// fixedHeight returns the number of lines consumed by non-output panels.
func (m Model) fixedHeight() int {
	return 5 // input(1) + dashboard(1) + separators(2) + padding(1)
}

// switchRole transitions to a new staff role — swaps prompt, mode, and narrates.
func (m *Model) switchRole(roleName string) {
	role, ok := m.roles[roleName]
	if !ok {
		return
	}

	m.currentRole = roleName
	m.systemPrompt = role.Prompt
	m.chatDriver.SetSystemPrompt(role.Prompt)

	if newMode, err := agent.ParseMode(role.Mode); err == nil {
		m.mode = newMode
	}

	m.dashboard.SetUIState(strings.ToUpper(roleName))
	m.dashboard.SetIdentity(m.sess.Workspace, m.sess.Driver, m.sess.Model, m.mode.String())
	m.roleMemory.AppendBriefing(staff.Entry{
		Content: fmt.Sprintf("→ switched to %s", roleName),
	})
	m.outputPanel.Append(tui.DimStyle.Render(
		fmt.Sprintf("  → %s", roleName)))
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
				_, cmd = m.inputPanel.Update(msg)
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

	// Tab: try slash-command completion first, then cycle focus
	if msg.Type == tea.KeyTab && m.state == stateInput {
		if m.inputPanel.TabComplete() {
			return m, nil
		}
		m.focus.Cycle()
		return m, nil
	}

	// Forward PgUp/PgDn to output panel for scrolling
	if msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown {
		_, cmd := m.outputPanel.Update(msg)
		return m, cmd
	}

	if m.state == stateInput {
		// Alt+M cycles mode: ask → plan → agent → auto → ask
		if msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'm' || msg.Runes[0] == 'M') {
			m.mode = m.mode.Next()
			m.sess.Mode = m.mode.String()
			m.outputPanel.Append(tui.DimStyle.Render(fmt.Sprintf("  mode: %s", m.mode)))
			return m, nil
		}

		// Input history navigation
		switch msg.Type {
		case tea.KeyUp:
			m.inputPanel.HistoryUp()
			return m, nil
		case tea.KeyDown:
			m.inputPanel.HistoryDown()
			return m, nil
		}

		var cmd tea.Cmd
		_, cmd = m.inputPanel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.inputPanel.Value())
	m.inputPanel.Reset()

	if input == "" {
		return m, nil
	}

	// Record in input history
	m.inputPanel.AddHistory(input)

	// Add user input to conversation
	m.outputPanel.Append(tui.UserStyle.Render(tui.LabelUser) + input)

	// Handle /role before the general command dispatcher (needs Model access)
	if cmd, ok := ParseCommand(input); ok && cmd.Name == "/role" {
		if len(cmd.Args) == 0 {
			names := make([]string, 0, len(m.roles))
			for n := range m.roles {
				names = append(names, n)
			}
			sort.Strings(names)
			m.outputPanel.Append(fmt.Sprintf("current role: %s\navailable: %s",
				m.currentRole, strings.Join(names, ", ")))
		} else if cmd.Args[0] == "create" && len(cmd.Args) >= 3 {
			// /role create <name> <mode> — create a role on the fly
			name, mode := cmd.Args[1], cmd.Args[2]
			m.roles[name] = staff.Role{
				Name:   name,
				Prompt: fmt.Sprintf("You are %s. The operator created this role on the fly.", name),
				Mode:   mode,
				Slots:  []string{}, // empty until configured
			}
			m.outputPanel.Append(fmt.Sprintf("created role %q (mode: %s, slots: none — use /role slots %s to configure)", name, mode, name))
		} else {
			m.switchRole(cmd.Args[0])
			m.outputPanel.Append(fmt.Sprintf("switched to %s (manual override)", cmd.Args[0]))
		}
		m.outputPanel.Append("")
		return m, nil
	}

	// Check for slash command
	if cmd, ok := ParseCommand(input); ok {
		result := ExecuteCommand(cmd, m.sess)
		if result.Output != "" {
			m.outputPanel.Append(result.Output)
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
		m.outputPanel.Append("") // blank line
		return m, nil
	}

	// Start agent loop
	m.state = stateStreaming
	m.dashboard.SetUIState("STREAMING")
	m.streamBuf.Reset()
	m.lastUsage = nil
	m.lastError = ""
	m.spinnerActive = true
	m.outputPanel.Append(tui.AssistStyle.Render(tui.LabelAssist) + ": ")
	m.inputPanel.BlurInput()

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
	m.dashboard.SetUIState("STREAMING")

	if approved {
		m.outputPanel.Append(tui.DimStyle.Render("  approved"))
	} else {
		m.outputPanel.Append(tui.ErrorStyle.Render("  denied"))
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
	text := m.streamBuf.String()
	m.streamBuf.Reset()

	// Accumulate raw text for completion-time markdown render.
	m.rawStreamLine.WriteString(text)

	// Display raw text during streaming — glamour renders on completion only.
	// Glamour's width-padding ANSI codes look like garbage inside lipgloss borders.
	m.outputPanel.AppendToLast(text)
}

// Test accessors — exported for acceptance tests.

// renderMOTD builds the welcome banner with logo and workspace info
// inside a lipgloss rounded border box.
func renderMOTD(sess *session.Session, tools *builtin.Registry, version, currentRole string, width int) string {
	logo := tui.LogoStyle.Render(tui.DjinnLogo)

	wsName := sess.Workspace
	if wsName == "" {
		wsName = "(ephemeral)"
	}

	if version == "" {
		version = "dev"
	}

	mode := sess.Mode
	if mode == "" {
		mode = "agent"
	}

	var info strings.Builder
	fmt.Fprintf(&info, "  ecosystem: %s\n", wsName)
	fmt.Fprintf(&info, "  model:     %s\n", sess.Model)
	fmt.Fprintf(&info, "  mode:      %s\n", mode)
	fmt.Fprintf(&info, "  role:      %s\n", currentRole)
	fmt.Fprintf(&info, "  tools:     %d built-in\n", len(tools.Names()))
	info.WriteString("\n")
	info.WriteString(tui.DimStyle.Render("  /help for commands"))
	info.WriteString("\n")
	info.WriteString(tui.DimStyle.Render("  Tab to complete slash commands"))

	inner := lipgloss.JoinHorizontal(lipgloss.Top, logo+"   ", info.String())

	// Bordered box with version in top border.
	boxWidth := width - 2
	if boxWidth < 40 {
		boxWidth = 40
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.RedHatRed).
		Padding(1, 2).
		Width(boxWidth)

	title := tui.AssistStyle.Render(" Djinn v" + version + " ")

	return box.Render(title + "\n\n" + inner)
}

// SetTextInput sets the text input value (for testing).
func (m *Model) SetTextInput(v string) { m.inputPanel.SetValue(v) }

// SetState sets the model state (for testing).
func (m *Model) SetState(s State) { m.state = s }

// SetMode overrides the agent mode (for testing tool approval flows).
func (m *Model) SetMode(mode agent.Mode) { m.mode = mode }

// AppendConversation adds a line to conversation (for testing).
func (m *Model) AppendConversation(line string) { m.outputPanel.Append(line) }

// ConversationLen returns the number of conversation lines (for testing).
func (m *Model) ConversationLen() int { return m.outputPanel.LineCount() }

// StreamBufString returns the stream buffer contents (for testing).
func (m Model) StreamBufString() string { return m.streamBuf.String() }

// CurrentState returns the current state (for testing).
func (m Model) CurrentState() State { return m.state }

// TextInputValue returns the text input value (for testing).
func (m Model) TextInputValue() string { return m.inputPanel.Value() }

// AddInputHistory adds an entry to input history (for testing).
func (m *Model) AddInputHistory(s string) { m.inputPanel.AddHistory(s) }

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
