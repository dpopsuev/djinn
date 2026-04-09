// scripted_driver.go — ScriptedChatDriver: scripts multi-turn conversations
// with tool calls for testing agent.Run()'s tool execution path.
//
// Each ScriptedTurn defines what the "LLM" responds with on that turn:
// text, tool calls, or both. Chat() returns the next turn's events.
// SendRich() records tool results for assertion.
package stubs

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// ScriptedTurn defines one turn of a scripted conversation.
type ScriptedTurn struct {
	Text      string            // text response (emitted as EventText)
	ToolCalls []driver.ToolCall // tool calls (emitted as EventToolUse)
	Usage     *driver.Usage     // token usage (emitted with EventDone)
}

// ScriptedChatDriver implements driver.ChatDriver with per-turn scripted responses.
// Supports text + tool calls. Records all Send/SendRich/AppendAssistant calls.
type ScriptedChatDriver struct {
	mu           sync.Mutex
	turns        []ScriptedTurn
	currentTurn  int
	systemPrompt string

	// Recorded interactions for assertion.
	SendLog     []driver.Message
	SendRichLog []driver.RichMessage
	HistoryLog_ []driver.RichMessage
}

var _ driver.ChatDriver = (*ScriptedChatDriver)(nil)

// NewScriptedChatDriver creates a ChatDriver that replays scripted turns.
func NewScriptedChatDriver(turns ...ScriptedTurn) *ScriptedChatDriver {
	return &ScriptedChatDriver{turns: turns}
}

func (d *ScriptedChatDriver) Start(_ context.Context, _ driver.SandboxHandle) error { return nil }
func (d *ScriptedChatDriver) Stop(_ context.Context) error                          { return nil }

func (d *ScriptedChatDriver) Send(_ context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.SendLog = append(d.SendLog, msg)
	return nil
}

func (d *ScriptedChatDriver) SendRich(_ context.Context, msg driver.RichMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.SendRichLog = append(d.SendRichLog, msg)
	return nil
}

// Chat returns the next scripted turn as a stream of events.
// Each call advances to the next turn. If exhausted, returns text="(no more turns)".
func (d *ScriptedChatDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	turn := d.currentTurn
	d.currentTurn++
	d.mu.Unlock()

	ch := make(chan driver.StreamEvent, 20)
	go func() {
		defer close(ch)

		if turn >= len(d.turns) {
			ch <- driver.StreamEvent{Type: driver.EventText, Text: "(no more turns)"}
			ch <- driver.StreamEvent{Type: driver.EventDone, Usage: &driver.Usage{}}
			return
		}

		t := d.turns[turn]

		// Emit text if present
		if t.Text != "" {
			ch <- driver.StreamEvent{Type: driver.EventText, Text: t.Text}
		}

		// Emit tool calls
		for i := range t.ToolCalls {
			ch <- driver.StreamEvent{
				Type:     driver.EventToolUse,
				ToolCall: &t.ToolCalls[i],
			}
		}

		// Emit done
		usage := t.Usage
		if usage == nil {
			usage = &driver.Usage{OutputTokens: 10}
		}
		ch <- driver.StreamEvent{Type: driver.EventDone, Usage: usage}
	}()

	return ch, nil
}

func (d *ScriptedChatDriver) AppendAssistant(msg driver.RichMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.HistoryLog_ = append(d.HistoryLog_, msg)
}

func (d *ScriptedChatDriver) SetSystemPrompt(prompt string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.systemPrompt = prompt
}

func (d *ScriptedChatDriver) ContextWindow() int { return 200_000 }

// --- Assertion helpers ---

// TurnCount returns how many turns were consumed.
func (d *ScriptedChatDriver) TurnCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.currentTurn
}

// ToolResults returns all tool result blocks sent via SendRich.
func (d *ScriptedChatDriver) ToolResults() []driver.ToolResult {
	d.mu.Lock()
	defer d.mu.Unlock()
	var results []driver.ToolResult
	for _, msg := range d.SendRichLog {
		for _, b := range msg.Blocks {
			if b.Type == driver.BlockToolResult && b.ToolResult != nil {
				results = append(results, *b.ToolResult)
			}
		}
	}
	return results
}

// MustJSON marshals v to json.RawMessage, panics on error.
func MustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
