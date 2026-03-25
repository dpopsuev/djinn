// scripted.go — ScriptedDriver replays scripted conversations for E2E testing.
// Supports streaming text, thinking blocks, tool calls, and multi-turn
// conversations where tool results trigger the next step.
package stubs

import (
	"context"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// ScriptedStep defines one turn of agent behavior.
type ScriptedStep struct {
	TextDeltas []string         // streamed text chunks (each becomes EventText)
	Thinking   string           // thinking block (EventThinking)
	ToolCall   *driver.ToolCall // request a tool call (EventToolUse)
	Usage      *driver.Usage    // usage stats sent with EventDone
}

// ScriptedDriver replays a scripted conversation for deterministic E2E testing.
// Each call to Chat() replays the next step. When the agent receives a tool
// result via Send/SendRich, the next Chat() call replays the following step.
type ScriptedDriver struct {
	mu           sync.Mutex
	steps        []ScriptedStep
	stepIdx      int
	started      bool
	stopped      bool
	systemPrompt string
	window       int
	SendLog      []driver.Message
	SendRichLog  []driver.RichMessage
}

// NewScriptedDriver creates a driver that replays the given steps in order.
func NewScriptedDriver(steps ...ScriptedStep) *ScriptedDriver {
	return &ScriptedDriver{
		steps:  steps,
		window: 200_000,
	}
}

// WithContextWindow sets the context window for the scripted driver.
func (d *ScriptedDriver) WithContextWindow(tokens int) *ScriptedDriver {
	d.window = tokens
	return d
}

func (d *ScriptedDriver) Start(_ context.Context, _ driver.SandboxHandle) error {
	d.started = true
	return nil
}

func (d *ScriptedDriver) Stop(_ context.Context) error {
	d.stopped = true
	return nil
}

func (d *ScriptedDriver) Send(_ context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.SendLog = append(d.SendLog, msg)
	return nil
}

func (d *ScriptedDriver) SendRich(_ context.Context, msg driver.RichMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.SendRichLog = append(d.SendRichLog, msg)
	return nil
}

func (d *ScriptedDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	idx := d.stepIdx
	if idx >= len(d.steps) {
		d.mu.Unlock()
		// No more steps — send done immediately.
		ch := make(chan driver.StreamEvent, 1)
		ch <- driver.StreamEvent{Type: driver.EventDone}
		close(ch)
		return ch, nil
	}
	step := d.steps[idx]
	d.stepIdx++
	d.mu.Unlock()

	ch := make(chan driver.StreamEvent, len(step.TextDeltas)+3)
	go func() {
		defer close(ch)

		// Thinking first.
		if step.Thinking != "" {
			ch <- driver.StreamEvent{Type: driver.EventThinking, Thinking: step.Thinking}
		}

		// Stream text deltas.
		for _, delta := range step.TextDeltas {
			ch <- driver.StreamEvent{Type: driver.EventText, Text: delta}
		}

		// Tool call.
		if step.ToolCall != nil {
			ch <- driver.StreamEvent{Type: driver.EventToolUse, ToolCall: step.ToolCall}
		}

		// Done with usage.
		done := driver.StreamEvent{Type: driver.EventDone}
		if step.Usage != nil {
			done.Usage = step.Usage
		}
		ch <- done
	}()

	return ch, nil
}

func (d *ScriptedDriver) AppendAssistant(_ driver.RichMessage) {}

func (d *ScriptedDriver) SetSystemPrompt(prompt string) {
	d.systemPrompt = prompt
}

func (d *ScriptedDriver) ContextWindow() int {
	return d.window
}

// StepIndex returns the current step index (for assertions).
func (d *ScriptedDriver) StepIndex() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stepIdx
}

// Started returns whether Start was called.
func (d *ScriptedDriver) Started() bool { return d.started }

// Stopped returns whether Stop was called.
func (d *ScriptedDriver) Stopped() bool { return d.stopped }

var _ driver.ChatDriver = (*ScriptedDriver)(nil)
