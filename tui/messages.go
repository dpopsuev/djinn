package tui

import (
	"time"

	"github.com/dpopsuev/djinn/driver"
)

// Custom tea.Msg types for bridging agent events to Bubbletea.

// TextMsg carries streamed text from the agent.
type TextMsg string

// ThinkingMsg carries streamed thinking from the agent.
type ThinkingMsg string

// ToolCallMsg signals the agent wants to call a tool.
type ToolCallMsg struct {
	Call driver.ToolCall
}

// ToolResultMsg carries the result of a tool execution.
type ToolResultMsg struct {
	CallID  string
	Name    string
	Output  string
	IsError bool
}

// DoneMsg signals the agent turn is complete.
type DoneMsg struct {
	Usage *driver.Usage
}

// ErrorMsg signals an error from the agent.
type ErrorMsg struct {
	Err error
}

func (e ErrorMsg) Error() string { return e.Err.Error() }

// AgentDoneMsg signals the full agent.Run() goroutine completed.
type AgentDoneMsg struct {
	Result string
	Err    error
}

// TickMsg triggers a render cycle for smooth streaming.
type TickMsg time.Time

// --- Panel messages: each panel receives state via messages, not method calls. ---

// OutputPanel messages.
type OutputAppendMsg struct{ Line string }
type OutputSetLineMsg struct{ Index int; Line string }
type OutputAppendLastMsg struct{ Text string }
type OutputClearMsg struct{}
type OutputSetOverlayMsg struct{ Text string }
type OutputCommitMsg struct{} // move pending lines to committed (two-zone)
type FlushStreamMsg struct{}  // flush stream buffer to last line

// ThinkingPanel messages.
type ThinkingClearMsg struct{}

// CommandsPanel messages.
type CommandsShowMsg struct{ Filter string }
type CommandsHideMsg struct{}

// InputPanel messages.
type InputSetValueMsg struct{ Value string }
type InputResetMsg struct{}
type InputFocusMsg struct{}
type InputBlurMsg struct{}
type InputAddHistoryMsg struct{ Value string }
type InputSetCompletionsMsg struct{ Names []string }

// InputSetPlaceholderMsg sets the input placeholder text.
type InputSetPlaceholderMsg struct{ Text string }

// SubmitMsg is emitted by InputPanel when the user presses Enter.
type SubmitMsg struct{ Value string }

// DashboardPanel messages.
type DashboardIdentityMsg struct{ Workspace, Driver, Model, Mode string }
type DashboardMetricsMsg struct{ TokensIn, TokensOut, Turns int }
type DashboardHealthMsg struct{ Reports []HealthReport }
type DashboardUIStateMsg struct{ State string }

// Layout messages.
type ResizeMsg struct{ Width, Height int }
type FocusPanelMsg struct{ Index int }
