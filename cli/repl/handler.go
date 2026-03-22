package repl

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dpopsuev/djinn/driver"
)

// BubbletaHandler bridges agent.EventHandler to Bubbletea messages.
// Each On* method sends a tea.Msg via program.Send(), which is
// thread-safe and non-blocking.
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
}

func (h *BubbletaHandler) OnDone(usage *driver.Usage) {
	h.program.Send(DoneMsg{Usage: usage})
}

func (h *BubbletaHandler) OnError(err error) {
	h.program.Send(ErrorMsg{Err: err})
}
