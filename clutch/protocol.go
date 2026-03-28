// Package clutch implements the shell/backend split protocol.
// The shell (TUI) and backend (agent runtime) communicate via
// typed messages over a transport (in-memory channels for MVP,
// Unix socket for production). Named after the clutch in a manual
// transmission: disengage, swap engine, re-engage.
package clutch

import (
	"encoding/json"

	"github.com/dpopsuev/djinn/driver"
)

// Protocol version.
const ProtocolVersion = 1

// Message types from Shell → Backend.
const (
	ShellPrompt   = "prompt"
	ShellApproval = "approval"
	ShellCommand  = "command"
	ShellQuit     = "quit"
)

// Message types from Backend → Shell.
const (
	BackendText       = "text"
	BackendThinking   = "thinking"
	BackendToolCall   = "tool_call"
	BackendToolResult = "tool_result"
	BackendDone       = "done"
	BackendError      = "error"
	BackendState      = "session_state"
	BackendReady      = "ready"
	BackendExiting    = "exiting"
)

// ShellMsg is a message from the shell to the backend.
type ShellMsg struct {
	Type       string   `json:"type"`
	Text       string   `json:"text,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
	Approved   bool     `json:"approved,omitempty"`
	CmdName    string   `json:"cmd_name,omitempty"`
	CmdArgs    []string `json:"cmd_args,omitempty"`
}

// BackendMsg is a message from the backend to the shell.
type BackendMsg struct {
	Type       string           `json:"type"`
	Text       string           `json:"text,omitempty"`
	ToolCall   *driver.ToolCall `json:"tool_call,omitempty"`
	ToolName   string           `json:"tool_name,omitempty"`
	ToolOutput string           `json:"tool_output,omitempty"`
	IsError    bool             `json:"is_error,omitempty"`
	Usage      *driver.Usage    `json:"usage,omitempty"`
	Error      string           `json:"error,omitempty"`
	Model      string           `json:"model,omitempty"`
	HistoryLen int              `json:"history_len,omitempty"`
	Version    int              `json:"version,omitempty"`
}

// Transport abstracts the communication channel between shell and backend.
// MVP: in-memory channels. Production: Unix socket with JSON lines.
type Transport interface {
	// Shell side
	SendToBackend(msg ShellMsg) error
	RecvFromBackend() (BackendMsg, error)

	// Backend side
	SendToShell(msg BackendMsg) error
	RecvFromShell() (ShellMsg, error)

	Close() error
}

// Hub registration — clients identify their role when connecting to the hub.
const HubRegister = "register"

// RegisterMsg is the first message a client sends to the hub.
type RegisterMsg struct {
	Role string `json:"role"` // "shell" or "backend"
}

// ensure json is importable for future socket transport
var _ = json.Marshal
