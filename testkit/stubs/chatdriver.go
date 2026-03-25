// chatdriver.go — stub ChatDriver for testing REPL, clutch, and agent integration.
package stubs

import (
	"context"

	"github.com/dpopsuev/djinn/driver"
)

// StubChatDriver implements driver.ChatDriver with canned streaming responses.
type StubChatDriver struct {
	model        string
	systemPrompt string
	messages     []driver.Message
	history      []driver.RichMessage
	SendLog      []driver.Message     // records all Send calls
	SendRichLog  []driver.RichMessage // records all SendRich calls
}

// NewStubChatDriver creates a ChatDriver that returns canned responses.
func NewStubChatDriver(responses ...driver.Message) *StubChatDriver {
	return &StubChatDriver{messages: responses}
}

func (d *StubChatDriver) Start(_ context.Context, _ driver.SandboxHandle) error { return nil }
func (d *StubChatDriver) Stop(_ context.Context) error                          { return nil }

func (d *StubChatDriver) Send(_ context.Context, msg driver.Message) error {
	d.SendLog = append(d.SendLog, msg)
	return nil
}

func (d *StubChatDriver) SendRich(_ context.Context, msg driver.RichMessage) error {
	d.SendRichLog = append(d.SendRichLog, msg)
	return nil
}

func (d *StubChatDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	ch := make(chan driver.StreamEvent, 10)
	go func() {
		defer close(ch)
		for _, m := range d.messages {
			ch <- driver.StreamEvent{Type: driver.EventText, Text: m.Content}
		}
		ch <- driver.StreamEvent{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 10}}
	}()
	return ch, nil
}

func (d *StubChatDriver) AppendAssistant(msg driver.RichMessage) {
	d.history = append(d.history, msg)
}

// HistoryLog returns all AppendAssistant calls for test assertions.
func (d *StubChatDriver) HistoryLog() []driver.RichMessage {
	return d.history
}

func (d *StubChatDriver) SetSystemPrompt(prompt string) {
	d.systemPrompt = prompt
}

// ContextWindow returns the model's context window in tokens.
func (d *StubChatDriver) ContextWindow() int { return 200_000 }

// Ensure StubChatDriver satisfies driver.ChatDriver.
var _ driver.ChatDriver = (*StubChatDriver)(nil)
