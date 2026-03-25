package repl

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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
	"github.com/dpopsuev/djinn/vcs"
)

// Frame rate for streaming — 20fps is smooth enough for text, reduces jitter.
const tickInterval = 50 * time.Millisecond

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
	state        State
	inputPanel   *tui.InputPanel
	pendingTool  *driver.ToolCall
	queuePanel   *tui.QueuePanel // visible queue between output and input
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
	envelopes      map[int]*tui.EnvelopePanel // tool envelopes keyed by output line index
	outputPanel    *tui.OutputPanel
	thinkingPanel  *tui.ThinkingPanel
	commandsPanel  *tui.CommandsPanel
	dashboard      *tui.DashboardPanel
	focus          *tui.FocusManager
	layout         *tui.LayoutEngine
	ready          bool
	quitting       bool
	initialPrompt  string // auto-submit on first render
	version        string // app version for MOTD
	rawStreamLine  *strings.Builder // raw unrendered text for incremental markdown (pointer: Builder can't be copied)

	// Tool routing
	router       *staff.SlotRouter // slot-filtered tool dispatch (nil = raw registry)

	// Worktree isolation for executor tasks
	worktreeMgr  *vcs.WorktreeManager
	activeWorktree string // current worktree path (empty = main repo)

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
	if placeholder := pickPlaceholderFile(cfg.Session.WorkDirs); placeholder != "" {
		inputPanel.Update(tui.InputSetPlaceholderMsg{Text: fmt.Sprintf("Try \"explain %s\"", placeholder)})
	}

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
		spin: spinner.New(
			spinner.WithSpinner(spinner.Spinner{
				Frames: tui.SpinnerFrames,
				FPS:    150 * time.Millisecond,
			}),
			spinner.WithStyle(tui.LogoStyle),
		),
		activeToolIdx:  -1,
		outputPanel:    tui.NewOutputPanel(),
		thinkingPanel:  tui.NewThinkingPanel(),
		queuePanel:     tui.NewQueuePanel(),
		commandsPanel:  tui.NewCommandsPanel(CommandNames()),
		dashboard:      tui.NewDashboardPanel(),
		initialPrompt:  cfg.InitialPrompt,
		version:        cfg.Version,
		router:         cfg.Router,
		worktreeMgr:    vcs.NewWorktreeManager(cfg.Session.WorkDir),
		debugTap:       cfg.DebugTap,
		chunkedBuf:     &strings.Builder{},
		rawStreamLine:  &strings.Builder{},
	}

	m.focus = tui.NewFocusManager(m.outputPanel, m.inputPanel, m.dashboard)
	m.focus.FocusPanel(1) // Default focus on input — user can type immediately.

	// LayoutEngine — declarative panel composition (SPC-52).
	m.layout = tui.NewLayoutEngine(m.focus)
	m.layout.Register(tui.PanelSlot{Panel: m.outputPanel, Weight: 1, MinHeight: 3, Border: tui.BorderOnly, Focusable: true})
	m.layout.Register(tui.PanelSlot{Panel: m.thinkingPanel, Visible: func() bool { return m.thinkingPanel.Active() }, Border: tui.BorderNone})
	m.layout.Register(tui.PanelSlot{Panel: m.queuePanel, Visible: func() bool { return m.queuePanel.Len() > 0 }, Border: tui.BorderFocusDepth, Focusable: true})
	m.layout.Register(tui.PanelSlot{Panel: m.inputPanel, Border: tui.BorderFocusDepth, Focusable: true})
	m.layout.Register(tui.PanelSlot{Panel: m.commandsPanel, Visible: func() bool { return m.commandsPanel.Active() }, Border: tui.BorderNone})
	m.layout.Register(tui.PanelSlot{Panel: m.dashboard, Border: tui.BorderFocusDepth, Focusable: true})
	m.dashboard.Update(tui.DashboardIdentityMsg{Workspace: cfg.Session.Workspace, Driver: cfg.Session.Driver, Model: cfg.Session.Model, Mode: cfg.Mode})
	m.dashboard.Update(tui.DashboardHealthMsg{Reports: cfg.HealthReports})

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
	m.dashboard.Update(tui.DashboardIdentityMsg{Workspace: cfg.Session.Workspace, Driver: cfg.Session.Driver, Model: cfg.Session.Model, Mode: m.mode.String()})
	m.dashboard.Update(tui.DashboardUIStateMsg{State: "GENSEC"})

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
		// LayoutEngine handles panel resize in Render().
		if m.outputPanel.LineCount() == 0 {
			m.outputPanel.Update(tui.OutputAppendMsg{Line: renderMOTD(m.sess, m.tools, m.version, m.currentRole)})
		}
		if m.initialPrompt != "" {
			m.inputPanel.Update(tui.InputSetValueMsg{Value: m.initialPrompt})
			m.initialPrompt = ""
			return m.handleSubmit()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tui.SubmitMsg:
		// InputPanel emitted a submit. Queue during streaming, process during input.
		if m.state == stateStreaming || m.state == stateToolApproval {
			m.queuePanel.Update(tui.QueueAddMsg{Prompt: msg.Value})
			return m, nil
		}
		// Not streaming — submit directly.
		m.inputPanel.Update(tui.InputAddHistoryMsg{Value: msg.Value})
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.UserStyle.Render(tui.LabelUser) + msg.Value})
		m.state = stateStreaming
		m.dashboard.Update(tui.DashboardUIStateMsg{State: "STREAMING"})
		m.lastUsage = nil
		m.lastError = ""
		m.spinnerActive = true
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.AssistStyle.Render(tui.LabelAssist) + ": "})
		return m, tea.Batch(m.runAgent(msg.Value), m.spin.Tick, tickCmd())

	case spinner.TickMsg:
		if m.spinnerActive {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case tui.TextMsg:
		m.spinnerActive = false
		if m.outputMode == outputChunked {
			m.chunkedBuf.WriteString(string(msg))
		} else {
			m.outputPanel.Update(msg) // OutputPanel handles TextMsg via streamBuf
		}
		return m, nil

	case tui.ThinkingMsg:
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render(string(msg))})
		return m, nil

	case tui.ToolCallMsg:
		envID := fmt.Sprintf("tool-%d", m.outputPanel.LineCount())
		env := tui.NewEnvelopePanel(envID, msg.Call.Name, string(msg.Call.Input))
		m.outputPanel.Update(tui.OutputAppendMsg{Line: env.View(m.width)})
		m.activeToolIdx = m.outputPanel.LineCount() - 1
		if m.envelopes == nil {
			m.envelopes = make(map[int]*tui.EnvelopePanel)
		}
		m.envelopes[m.activeToolIdx] = env

		if m.mode == agent.ModeAgent && !m.autoApprove {
			m.state = stateToolApproval
			m.dashboard.Update(tui.DashboardUIStateMsg{State: "APPROVAL"})
			m.pendingTool = &msg.Call
		}
		return m, nil

	case tui.ToolResultMsg:
		if m.activeToolIdx >= 0 && m.activeToolIdx < m.outputPanel.LineCount() {
			if env, ok := m.envelopes[m.activeToolIdx]; ok {
				env.SetResult(msg.Output, msg.IsError)
				m.outputPanel.Update(tui.OutputSetLineMsg{Index: m.activeToolIdx, Line: env.View(m.width)})
			}
			m.activeToolIdx = -1
		} else {
			var line string
			if msg.IsError {
				line = fmt.Sprintf("  %s %s",
					tui.ErrorStyle.Render(tui.GlyphToolError+" "+msg.Name),
					tui.DimStyle.Render(truncate(msg.Output, 100)))
			} else {
				line = fmt.Sprintf("  %s %s",
					tui.ToolSuccessStyle.Render(tui.GlyphToolSuccess+" "+msg.Name),
					tui.DimStyle.Render(truncate(msg.Output, 100)))
			}
			m.outputPanel.Update(tui.OutputAppendMsg{Line: line})
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
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render("error: " + msg.Error())})
		return m, nil

	case tui.AgentDoneMsg:
		m.rawStreamLine.Reset()
		if m.store != nil {
			if err := m.store.Save(m.sess); err != nil {
				m.log.Warn("auto-save failed", "error", err)
			}
		}
		// Flush remaining buffers
		if m.outputMode == outputChunked && m.chunkedBuf.Len() > 0 {
			if m.outputPanel.LineCount() > 0 {
				last := m.outputPanel.LineCount() - 1
				m.outputPanel.Update(tui.OutputSetLineMsg{Index: last, Line: m.outputPanel.Lines()[last] + m.chunkedBuf.String()})
			}
			m.chunkedBuf.Reset()
		}
		m.outputPanel.Update(tui.FlushStreamMsg{})
		// Render completed response as markdown
		if m.outputPanel.LineCount() > 0 {
			last := m.outputPanel.LineCount() - 1
			raw := m.outputPanel.Lines()[last]
			prefix := tui.AssistStyle.Render(tui.LabelAssist) + ": "
			if after, found := strings.CutPrefix(raw, prefix); found {
				rendered := tui.RenderMarkdown(after)
				m.outputPanel.Update(tui.OutputSetLineMsg{Index: last, Line: prefix + rendered})
			}
		}
		if msg.Err != nil {
			m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render("error: " + msg.Err.Error())})
		}
		if m.lastUsage != nil {
			m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.StatusStyle.Render(fmt.Sprintf("[tokens: %d in, %d out]",
				m.lastUsage.InputTokens, m.lastUsage.OutputTokens))})
		}
		m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
		m.state = stateInput
		m.focus.FocusPanel(1)
		m.inputPanel.Update(tui.InputFocusMsg{})

		// Drain prompt queue — auto-submit first queued prompt.
		if m.queuePanel.Len() > 0 {
			next := m.queuePanel.Items()[0]
			m.queuePanel.Update(tui.QueueDrainMsg{})
			return m, func() tea.Msg { return tui.SubmitMsg{Value: next} }
		}

		// Auto-transition: executor gate check.
		if m.currentRole == "executor" {
			gate := &staff.MakeCircuitGate{}
			gateDir := m.sess.WorkDir
			if m.activeWorktree != "" {
				gateDir = m.activeWorktree
			}
			result, gateErr := gate.Check(m.ctx, gateDir)
			if gateErr != nil {
				m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render("gate error: " + gateErr.Error())})
			}
			if result.Passed {
				m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ToolSuccessStyle.Render("  ✓ gate passed")})
				next := staff.NextRole(staff.SignalGatePassed)
				m.switchRole(next)
			} else {
				for _, d := range result.Diagnostics {
					m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render(
						fmt.Sprintf("  ✗ %s: %s", d.Source, truncate(d.Message, 200)))})
				}
				m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("  gate failed — fix and try again")})
			}
		} else if m.currentRole != "gensec" {
			m.switchRole("gensec")
		} else {
			m.dashboard.Update(tui.DashboardUIStateMsg{State: "GENSEC"})
		}
		return m, nil

	case tui.TickMsg:
		if m.state == stateStreaming {
			m.outputPanel.Update(tui.FlushStreamMsg{})
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

	// Update panel state before render.
	m.outputPanel.Update(tui.OutputSetOverlayMsg{Text: m.overlayContent()})
	m.dashboard.Update(tui.DashboardMetricsMsg{TokensIn: m.totalIn, TokensOut: m.totalOut, Turns: m.sess.History.Len()})

	// LayoutEngine handles: visibility, heights, borders, focus sync.
	m.layout.Resize(m.width, m.height)
	result := m.layout.Render()

	// DebugTap: capture every rendered frame.
	if m.debugTap != nil {
		stateStr := "input"
		switch m.state {
		case stateStreaming:
			stateStr = "streaming"
		case stateToolApproval:
			stateStr = "approval"
		}
		innerWidth := m.width - 2
		m.debugTap.Capture(result, stateStr, m.currentRole, m.width, m.height, &tui.DebugComponents{
			Transcript: m.outputPanel.Lines(),
			Overlay:    m.overlayContent(),
			InputValue: m.inputPanel.Value(),
			InputFocus: m.inputPanel.Focused(),
			FocusedIdx: m.focus.ActiveIndex(),
			Dashboard:  m.dashboard.View(innerWidth),
		})
	}

	return result
}

// overlayContent returns ephemeral content for the output panel based on state.
func (m Model) overlayContent() string {
	switch {
	case m.state == stateStreaming && m.spinnerActive:
		return "  " + m.spin.View() + " thinking..."
	case m.state == stateStreaming && m.outputPanel.StreamBufString() != "":
		return m.outputPanel.StreamBufString()
	case m.state == stateToolApproval && m.pendingTool != nil:
		return tui.ToolNameStyle.Render(fmt.Sprintf("  approve %s? [y/n] ", m.pendingTool.Name))
	default:
		return ""
	}
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

	// Update tool restrictions based on role's allowed slots.
	m.token.AllowedTools = role.Slots
	if m.router != nil {
		m.router.SetRole(roleName)
	}

	m.dashboard.Update(tui.DashboardUIStateMsg{State: strings.ToUpper(roleName)})
	m.dashboard.Update(tui.DashboardIdentityMsg{Workspace: m.sess.Workspace, Driver: m.sess.Driver, Model: m.sess.Model, Mode: m.mode.String()})
	m.roleMemory.AppendBriefing(staff.Entry{
		Content: fmt.Sprintf("→ switched to %s", roleName),
	})
	m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render(
		fmt.Sprintf("  → %s", roleName))})
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

	// Tab: completion only (BUG-45). Focus cycling via Shift+Tab.
	if msg.Type == tea.KeyTab {
		if handled, cmd := m.inputPanel.TabComplete(); handled {
			return m, cmd
		}
		// If input has prediction, accept it.
		if m.inputPanel.AcceptPrediction() {
			return m, nil
		}
		return m, nil // no focus cycling on Tab
	}
	if msg.Type == tea.KeyShiftTab {
		m.focus.Cycle()
		return m, nil
	}

	// During streaming: Escape cancels the agent.
	if m.state == stateStreaming && msg.Type == tea.KeyEscape {
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("  (cancelled)")})
		m.outputPanel.Update(tui.FlushStreamMsg{})
		m.state = stateInput
		m.focus.FocusPanel(1)
		m.inputPanel.Update(tui.InputFocusMsg{})
		m.dashboard.Update(tui.DashboardUIStateMsg{State: "INSERT"})
		return m, nil
	}

	// Dive/Climb: Enter on non-input panel = Dive, Escape when dived = Climb.
	if msg.Type == tea.KeyEnter && m.focus.Active() != nil && m.focus.Active().ID() != "input" {
		if m.focus.Dive() {
			return m, nil
		}
	}
	if msg.Type == tea.KeyEscape && m.focus.Depth() > 0 {
		m.focus.Climb()
		return m, nil
	}

	// Forward PgUp/PgDn to output panel for scrolling (any state).
	if msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown {
		_, cmd := m.outputPanel.Update(msg)
		return m, cmd
	}

	// Alt+M cycles mode (any state — works during streaming too).
	if msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'm' || msg.Runes[0] == 'M') {
		m.mode = m.mode.Next()
		m.sess.Mode = m.mode.String()
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render(fmt.Sprintf("  mode: %s", m.mode))})
		return m, nil
	}

	// Input history navigation.
	switch msg.Type {
	case tea.KeyUp:
		m.inputPanel.HistoryUp()
		return m, nil
	case tea.KeyDown:
		m.inputPanel.HistoryDown()
		return m, nil
	}

	// Forward all other keys to input panel (type-ahead during streaming).
	var cmd tea.Cmd
	_, cmd = m.inputPanel.Update(msg)
	return m, cmd
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.inputPanel.Value())
	m.inputPanel.Update(tui.InputResetMsg{})

	if input == "" {
		return m, nil
	}

	m.inputPanel.Update(tui.InputAddHistoryMsg{Value: input})
	m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.UserStyle.Render(tui.LabelUser) + input})

	// Handle /role before the general command dispatcher (needs Model access)
	if cmd, ok := ParseCommand(input); ok && cmd.Name == "/role" {
		if len(cmd.Args) == 0 {
			names := make([]string, 0, len(m.roles))
			for n := range m.roles {
				names = append(names, n)
			}
			sort.Strings(names)
			m.outputPanel.Update(tui.OutputAppendMsg{Line: fmt.Sprintf("current role: %s\navailable: %s",
				m.currentRole, strings.Join(names, ", "))})
		} else if cmd.Args[0] == "create" && len(cmd.Args) >= 3 {
			name, mode := cmd.Args[1], cmd.Args[2]
			m.roles[name] = staff.Role{
				Name:   name,
				Prompt: fmt.Sprintf("You are %s. The operator created this role on the fly.", name),
				Mode:   mode,
				Slots:  []string{},
			}
			m.outputPanel.Update(tui.OutputAppendMsg{Line: fmt.Sprintf("created role %q (mode: %s, slots: none — use /role slots %s to configure)", name, mode, name)})
		} else {
			m.switchRole(cmd.Args[0])
			m.outputPanel.Update(tui.OutputAppendMsg{Line: fmt.Sprintf("switched to %s (manual override)", cmd.Args[0])})
		}
		m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
		return m, nil
	}

	// Handle /staff — show all roles and their current state.
	if cmd, ok := ParseCommand(input); ok && cmd.Name == "/staff" {
		var sb strings.Builder
		sb.WriteString("Staff:\n")
		names := make([]string, 0, len(m.roles))
		for n := range m.roles {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, name := range names {
			role := m.roles[name]
			indicator := "  "
			if name == m.currentRole {
				indicator = "→ "
			}
			sb.WriteString(fmt.Sprintf("%s%s (mode: %s, slots: %d)\n",
				indicator, name, role.Mode, len(role.Slots)))
		}
		m.outputPanel.Update(tui.OutputAppendMsg{Line: sb.String()})
		m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
		return m, nil
	}

	// Handle /briefing
	if cmd, ok := ParseCommand(input); ok && cmd.Name == "/briefing" {
		entries := m.roleMemory.Briefing()
		if len(entries) == 0 {
			m.outputPanel.Update(tui.OutputAppendMsg{Line: "briefing: (empty)"})
		} else {
			var sb strings.Builder
			sb.WriteString("Briefing:\n")
			for _, e := range entries {
				ts := e.Timestamp.Format("15:04:05")
				fmt.Fprintf(&sb, "  [%s] %s\n", ts, e.Content)
			}
			m.outputPanel.Update(tui.OutputAppendMsg{Line: sb.String()})
		}
		m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
		return m, nil
	}

	// Slash command dispatch
	if cmd, ok := ParseCommand(input); ok {
		result := ExecuteCommand(cmd, m.sess)
		if result.Output != "" {
			m.outputPanel.Update(tui.OutputAppendMsg{Line: result.Output})
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
		m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})
		return m, nil
	}

	// Start agent loop
	m.state = stateStreaming
	m.dashboard.Update(tui.DashboardUIStateMsg{State: "STREAMING"})
	m.lastUsage = nil
	m.lastError = ""
	m.spinnerActive = true
	m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.AssistStyle.Render(tui.LabelAssist) + ": "})
	// Input stays focused during streaming — enables type-ahead (BUG-44).

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
	m.dashboard.Update(tui.DashboardUIStateMsg{State: "STREAMING"})

	if approved {
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("  approved")})
	} else {
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render("  denied")})
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
		// Use slot router if available — filters tools by current role.
		// Falls back to raw registry if no router configured.
		var tools builtin.ToolExecutor = m.tools
		if m.router != nil {
			tools = m.router
		}
		result, err := agent.Run(m.ctx, agent.Config{
			Driver:       m.chatDriver,
			Tools:        tools,
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

// Test accessors — exported for acceptance tests.

// renderMOTD builds the welcome banner with logo and workspace info
// inside a lipgloss rounded border box.
func renderMOTD(sess *session.Session, tools *builtin.Registry, version, currentRole string) string {
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

	// No inner border — the panel border is the visual container.
	title := tui.AssistStyle.Render("Djinn v" + version)

	return title + "\n\n" + inner
}

// SetTextInput sets the text input value (for testing).
func (m *Model) SetTextInput(v string) { m.inputPanel.Update(tui.InputSetValueMsg{Value: v}) }

// SetState sets the model state (for testing).
func (m *Model) SetState(s State) { m.state = s }

// SetMode overrides the agent mode (for testing tool approval flows).
func (m *Model) SetMode(mode agent.Mode) { m.mode = mode }

// AppendConversation adds a line to conversation (for testing).
func (m *Model) AppendConversation(line string) { m.outputPanel.Update(tui.OutputAppendMsg{Line: line}) }

// ConversationLen returns the number of conversation lines (for testing).
func (m *Model) ConversationLen() int { return m.outputPanel.LineCount() }

// StreamBufString returns the stream buffer contents (for testing).
func (m Model) StreamBufString() string { return m.outputPanel.StreamBufString() }

// CurrentState returns the current state (for testing).
func (m Model) CurrentState() State { return m.state }

// TextInputValue returns the text input value (for testing).
func (m Model) TextInputValue() string { return m.inputPanel.Value() }

// AddInputHistory adds an entry to input history (for testing).
func (m *Model) AddInputHistory(s string) { m.inputPanel.Update(tui.InputAddHistoryMsg{Value: s}) }

// pickPlaceholderFile finds a Go file in the workspace dirs for the input placeholder.
func pickPlaceholderFile(dirs []string) string {
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				return e.Name()
			}
		}
	}
	return ""
}

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
