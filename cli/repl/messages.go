package repl

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
