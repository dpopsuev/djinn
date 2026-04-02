package tui

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dpopsuev/djinn/driver"
)

// TRUST BOUNDARY: Agent events → TUI messages.
//
// The handler is the trust boundary between the untrusted agent and the TUI.
// It ONLY emits output-safe messages:
//   - SAFE: TextMsg, ThinkingMsg, ToolCallMsg, ToolResultMsg, DoneMsg, ErrorMsg, RenderPanelMsg
//   - NEVER from agent: InputSetValueMsg, SubmitMsg, DialogResultMsg, FocusPanelMsg, ResizeMsg
//
// The agent cannot inject commands, modify input, trigger dialogs, or change layout.
// All agent output is confined to the output panel.

// MaxSightHintsPerTurn limits SightHintMsg per agent turn.
// Prevents agent from spamming highlight/pulse to distract operator.
// STRIDE T (Tampering) mitigation at trust boundary TB-1.
const MaxSightHintsPerTurn = 3

// BubbletaHandler bridges agent.EventHandler to Bubbletea messages.
type BubbletaHandler struct {
	program   *tea.Program
	hintCount int // SightHintMsg count in current turn
}

// NewHandler creates a handler that sends events to the given program.
func NewHandler(p *tea.Program) *BubbletaHandler {
	return &BubbletaHandler{program: p}
}

// EmitSightHint sends a SightHintMsg if under the per-turn rate limit.
// Excess hints are silently dropped. Counter resets on OnDone/OnError.
func (h *BubbletaHandler) EmitSightHint(hint SightHintMsg) bool {
	if h.hintCount >= MaxSightHintsPerTurn {
		return false // dropped
	}
	h.hintCount++
	h.program.Send(hint)
	return true
}

func (h *BubbletaHandler) OnText(text string) {
	h.program.Send(TextMsg(text))
}

func (h *BubbletaHandler) OnThinking(text string) {
	h.program.Send(ThinkingMsg(text))
}

func (h *BubbletaHandler) OnToolCall(call driver.ToolCall) {
	h.program.Send(ToolCallMsg{Call: call})
}

func (h *BubbletaHandler) OnToolResult(callID, name, output string, isError bool) {
	h.program.Send(ToolResultMsg{
		CallID:  callID,
		Name:    name,
		Output:  output,
		IsError: isError,
	})

	// Intercept render tool results — emit panel message to TUI.
	const renderToolName = "render"
	if name == renderToolName && !isError {
		var render struct {
			Type  string `json:"type"`
			Title string `json:"title"`
			Data  string `json:"data"`
		}
		if err := json.Unmarshal([]byte(output), &render); err == nil && render.Type != "" {
			h.program.Send(RenderPanelMsg{
				Type:  render.Type,
				Title: render.Title,
				Data:  render.Data,
			})
		}
	}
}

func (h *BubbletaHandler) OnDone(usage *driver.Usage) {
	h.hintCount = 0 // reset hint rate limit for next turn
	h.program.Send(DoneMsg{Usage: usage})
}

func (h *BubbletaHandler) OnError(err error) {
	h.hintCount = 0 // reset hint rate limit for next turn
	h.program.Send(ErrorMsg{Err: err})
}
