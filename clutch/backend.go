package clutch

import (
	"context"
	"fmt"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// BackendConfig configures the backend process.
type BackendConfig struct {
	Driver       *claudedriver.APIDriver
	Tools        *builtin.Registry
	Session      *session.Session
	SystemPrompt string
	MaxTurns     int
}

// RunBackend runs the backend loop: receives prompts from the shell,
// runs the agent, sends events back. Returns when the shell sends Quit
// or the context is cancelled.
func RunBackend(ctx context.Context, transport Transport, cfg BackendConfig) error {
	// Announce ready with session state
	transport.SendToShell(BackendMsg{
		Type:       BackendReady,
		Version:    ProtocolVersion,
		Model:      cfg.Session.Model,
		HistoryLen: cfg.Session.History.Len(),
	})

	for {
		msg, err := transport.RecvFromShell()
		if err != nil {
			return fmt.Errorf("recv from shell: %w", err)
		}

		switch msg.Type {
		case ShellQuit:
			transport.SendToShell(BackendMsg{Type: BackendExiting})
			return nil

		case ShellPrompt:
			handler := &backendHandler{transport: transport}
			result, err := agent.Run(ctx, agent.Config{
				Driver:       cfg.Driver,
				Tools:        cfg.Tools,
				Session:      cfg.Session,
				SystemPrompt: cfg.SystemPrompt,
				MaxTurns:     cfg.MaxTurns,
				Approve:      agent.AutoApprove,
				Handler:      handler,
			}, msg.Text)

			if err != nil {
				transport.SendToShell(BackendMsg{
					Type:  BackendError,
					Error: err.Error(),
				})
			}
			_ = result

		case ShellCommand:
			// Commands handled by shell directly for now
			// Future: some commands may need backend state

		case ShellApproval:
			// Approval handled inline by the agent loop
			// Future: wire to agent.ApprovalFunc via channel
		}
	}
}

// backendHandler implements agent.EventHandler and sends events
// to the shell via the transport.
type backendHandler struct {
	transport Transport
}

func (h *backendHandler) OnText(text string) {
	h.transport.SendToShell(BackendMsg{Type: BackendText, Text: text})
}

func (h *backendHandler) OnThinking(text string) {
	h.transport.SendToShell(BackendMsg{Type: BackendThinking, Text: text})
}

func (h *backendHandler) OnToolCall(call driver.ToolCall) {
	h.transport.SendToShell(BackendMsg{Type: BackendToolCall, ToolCall: &call})
}

func (h *backendHandler) OnToolResult(callID, name, output string, isError bool) {
	h.transport.SendToShell(BackendMsg{
		Type:       BackendToolResult,
		ToolName:   name,
		ToolOutput: output,
		IsError:    isError,
	})
}

func (h *backendHandler) OnDone(usage *driver.Usage) {
	h.transport.SendToShell(BackendMsg{Type: BackendDone, Usage: usage})
}

func (h *backendHandler) OnError(err error) {
	h.transport.SendToShell(BackendMsg{Type: BackendError, Error: err.Error()})
}
