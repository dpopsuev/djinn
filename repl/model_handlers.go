// model_handlers.go — Extracted Update handlers for the REPL model.
// Each method handles a group of related Bubbletea messages.
// Keeps the main Update switch thin (router, not handler).
package repl

import (
	"fmt"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea" //nolint:depguard // repl is TUI composition root

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tui"
	"github.com/dpopsuev/djinn/tui/elements"
	"github.com/dpopsuev/djinn/tui/widgets"
	"github.com/dpopsuev/djinn/uniform"
)

// handleSubmitMsg processes operator prompt submission.
func (m Model) handleSubmitMsg(msg tui.SubmitMsg) (tea.Model, tea.Cmd) { //nolint:gocritic // tea.Model requires value receiver
	// Queue during streaming, process during input.
	if m.state == stateStreaming || m.state == stateToolApproval {
		m.queuePanel.Update(tui.QueueAddMsg{Prompt: msg.Value})
		return m, nil
	}
	// Not streaming — submit directly.
	m.inputPanel.Update(tui.InputAddHistoryMsg(msg))
	m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.UserStyle.Render(tui.LabelUser) + msg.Value})
	m.state = stateStreaming
	m.dashboard.Update(tui.DashboardUIStateMsg{State: "STREAMING"})
	m.lastUsage = nil
	m.lastError = ""
	m.spinnerActive = true
	m.outputPanel.Update(tui.OutputAppendMsg{Line: ""})

	// Inject cell sight into prompt if the active panel supports it (GOL-62).
	prompt := msg.Value
	sightMgr := m.term.SightManager()
	if provider, ok := m.focus.Active().(tui.Sighted); ok {
		panelGate := provider.SightGate()
		mgrGate := sightMgr.IsGateOpen(provider.CellSight().PanelID)
		if panelGate && mgrGate {
			if cs := provider.CellSight(); !cs.IsEmpty() {
				cs = sightMgr.ApplyCellSight(cs)
				prompt = cs.FormatPrompt() + "\n\n" + prompt
			}
		}
	}
	return m, tea.Batch(m.runAgent(prompt), m.spin.Tick, tickCmd())
}

// handleStreamEvent processes real-time agent output events.
func (m Model) handleStreamEvent(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:gocritic // tea.Model requires value receiver
	switch msg := msg.(type) {
	case tui.TextMsg:
		m.spinnerActive = false
		m.isThinking = false
		if m.outputMode == outputChunked {
			m.chunkedBuf.WriteString(string(msg))
		} else {
			m.outputPanel.Update(msg)
		}

	case tui.ThinkingMsg:
		m.isThinking = true
		m.dashboard.Update(tui.DashboardUIStateMsg{State: "THINKING"})
		m.outputPanel.Update(tui.OutputAppendMsg{Line: elements.Dim(string(msg))})

	case tui.ToolCallMsg:
		envID := fmt.Sprintf("tool-%d", m.outputPanel.LineCount())
		env := widgets.NewEnvelopePanel(envID, msg.Call.Name, string(msg.Call.Input))
		m.outputPanel.Update(tui.OutputAppendMsg{Line: env.View(m.width)})
		m.activeToolIdx = m.outputPanel.LineCount() - 1
		if m.envelopes == nil {
			m.envelopes = make(map[int]*widgets.EnvelopePanel)
		}
		m.envelopes[m.activeToolIdx] = env

		if m.mode == agent.ModeAgent && !m.autoApprove {
			m.state = stateToolApproval
			m.dashboard.Update(tui.DashboardUIStateMsg{State: "APPROVAL"})
			m.pendingTool = &msg.Call
		}

	case tui.ToolResultMsg:
		if !msg.IsError && (msg.Name == "Write" || msg.Name == "Edit") {
			m.filesEdited++
		}
		if m.activeToolIdx >= 0 && m.activeToolIdx < m.outputPanel.LineCount() {
			if env, ok := m.envelopes[m.activeToolIdx]; ok {
				env.SetResult(msg.Output, msg.IsError)
				m.outputPanel.Update(tui.OutputSetLineMsg{Index: m.activeToolIdx, Line: env.View(m.width)})
			}
			m.activeToolIdx = -1
		} else {
			state := elements.StateDone
			if msg.IsError {
				state = elements.StateError
			}
			line := "  " + tui.ToolStatus(msg.Name, state, 0) + " " + tui.DimStyle.Render(truncate(msg.Output, 100))
			m.outputPanel.Update(tui.OutputAppendMsg{Line: line})
		}

	case tui.DoneMsg:
		m.lastUsage = msg.Usage
		if msg.Usage != nil {
			m.totalIn += msg.Usage.InputTokens
			m.totalOut += msg.Usage.OutputTokens
			m.monitor.Record(msg.Usage.InputTokens, msg.Usage.OutputTokens)
		}

	case tui.ErrorMsg:
		m.lastError = msg.Error()
		m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render("error: " + msg.Error())})
	}
	return m, nil
}

// handleAgentDone processes the completion of an agent run.
func (m Model) handleAgentDone(msg tui.AgentDoneMsg) (tea.Model, tea.Cmd) { //nolint:gocritic // tea.Model requires value receiver
	m.rawStreamLine.Reset()
	if m.store != nil {
		if err := m.store.Save(m.sess); err != nil {
			m.log.WarnContext(m.ctx, "auto-save failed", slog.String(telemetry.KeyError, err.Error()))
		}
	}
	// Flush remaining buffers.
	if m.outputMode == outputChunked && m.chunkedBuf.Len() > 0 {
		if m.outputPanel.LineCount() > 0 {
			last := m.outputPanel.LineCount() - 1
			m.outputPanel.Update(tui.OutputSetLineMsg{Index: last, Line: m.outputPanel.Lines()[last] + m.chunkedBuf.String()})
		}
		m.chunkedBuf.Reset()
	}
	m.outputPanel.Update(tui.FlushStreamMsg{})
	// Render completed response as markdown.
	if m.outputPanel.LineCount() > 0 {
		last := m.outputPanel.LineCount() - 1
		raw := m.outputPanel.Lines()[last]
		prefix := ""
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

	// Context relay: log approaching thresholds.
	if u := m.monitor.Usage(); u > 0.5 {
		m.log.InfoContext(m.ctx, "context usage", slog.String(telemetry.KeyStatus, fmt.Sprintf("%.0f%%", u*100)))
	}

	// Drain prompt queue — auto-submit first queued prompt.
	if m.queuePanel.Len() > 0 {
		next := m.queuePanel.Items()[0]
		m.queuePanel.Update(tui.QueueDrainMsg{})
		return m, func() tea.Msg { return tui.SubmitMsg{Value: next} }
	}

	// Auto-transition: role-based gate check.
	switch {
	case m.currentRole == roleExecutor:
		gate := &uniform.MakeCircuitGate{}
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
			next := uniform.NextRole(uniform.SignalGatePassed)
			m.switchRole(next)
		} else {
			for _, d := range result.Diagnostics {
				m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.ErrorStyle.Render(
					fmt.Sprintf("  ✗ %s: %s", d.Source, truncate(d.Message, 200)))})
			}
			m.outputPanel.Update(tui.OutputAppendMsg{Line: tui.DimStyle.Render("  gate failed — fix and try again")})
		}
	case m.currentRole != roleGensec:
		m.switchRole(roleGensec)
	default:
		m.dashboard.Update(tui.DashboardUIStateMsg{State: "GENSEC"})
	}
	return m, nil
}
