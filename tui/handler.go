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

// BubbletaHandler bridges agent.EventHandler to Bubbletea messages.
type BubbletaHandler struct {
	program *tea.Program
}

// NewHandler creates a handler that sends events to the given program.
func NewHandler(p *tea.Program) *BubbletaHandler {
	return &BubbletaHandler{program: p}
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
	if name == "render" && !isError {
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
	h.program.Send(DoneMsg{Usage: usage})
}

func (h *BubbletaHandler) OnError(err error) {
	h.program.Send(ErrorMsg{Err: err})
}
